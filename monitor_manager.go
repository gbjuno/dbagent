package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	//"reflect"
	"time"
)

var MonitorErr error = errors.New("monitor failed")

//MonitorManager is used for monitoring the mongoinstance and update the instance status on the apiserver
//During the startup of MongoAgent, MonitorManager will be initialized and run in thread.
type MonitorManager struct {
	ma      *MongoAgent
	insList []string
	join    chan string
	leave   chan string
}

func (mon *MonitorManager) monitorAll() {
	for {
		for _, insName := range mon.insList[:] {
			mon.simpleCheckOneIns(insName)
		}
		time.Sleep(60)
	}
}

func (mon *MonitorManager) monitorWorker(ctx context.Context, ins *Mongo) {

}

func (mon *MonitorManager) simpleCheckOneIns(insName string) {
	ins := mon.ma.mongoMap[insName]
	glog.Infof("checking mongo instance %s status...", ins.Name)
	conn, err := mgo.Dial(fmt.Sprintf("127.0.0.1:%d/admin", ins.Port))
	if err != nil {
		glog.Errorf("connect mongo instance %s failed", ins.Name)
		glog.Errorf("mongo instance %s is not running", ins.Name)
		ins.Running = false
		mon.ma.mongoMgr.Send(ins)
		return
	}
	defer conn.Close()
	c := conn.DB("local").C("startup_log")

	var result interface{}
	err = c.Find(nil).One(&result)
	if err != nil {
		glog.Errorf("cannot get data of DB local Table startup_log of mongo %s", ins.Name)
		glog.Errorf("mongo instance %s is not running property", ins.Name)
		ins.Running = false
		mon.ma.mongoMgr.Send(ins)
		return
	}
	glog.Infof("mongo instance %s is alive and ready", ins.Name)
	ins.Running = true
	glog.Infof("instance %v", ins)
	mon.ma.mongoMgr.Send(ins)
}

func (mon *MonitorManager) checkInsDetail(ctx context.Context, insNameCh <-chan string) {
	glog.Infof("monitor worker is running")
	for {
		select {
		case <-ctx.Done():
			glog.Infof("monitor worker exit because of ctx cancel")
			break
		case insName := <-insNameCh:
			glog.Infof("checking on mongo instance %s", insName)
			ins := mon.ma.mongoMap[insName]
			mon.getMongoStatus(ins)
		}
	}
	glog.Infof("monitor worker exits")
}

func (mon *MonitorManager) getMongoStatus(ins *Mongo) {
	conn, err := mgo.Dial(fmt.Sprintf("127.0.0.1:%d/admin", ins.Port))
	if err != nil {
		glog.Errorf("connect mongo instance %s failed", ins.Name)
		glog.Errorf("mongo instance %s is not running", ins.Name)
		ins.Running = false
		mon.ma.mongoMgr.Send(ins)
		return
	}
	var result bson.M
	err = conn.DB("admin").Run(bson.D{{"serverStatus", 1}}, &result)
	if err != nil {
		glog.Infof("get Mongo Server Status failed, err %c", err)
	}
	glog.Infof("Mongo Server Status: %v", result)
	defer conn.Close()
	j, err := json.Marshal(result["network"])
	if err != nil {
		glog.Errorf("cannot marshal network")
		return
	}
	v := &Network{}
	if err != nil {
		glog.Errorf("cannot unmarshal network")
		return
	}
	err = json.Unmarshal(j, &v)
	glog.Infof("network: %T %v", v, v)
	glog.Infof("network: %T %v", result["network"], result["network"])
	return
}

func (mon *MonitorManager) Register(insName string) {
	glog.Infof("register mongo instance %s into monitor list", insName)
	mon.join <- insName
	glog.Infof("register mongo instance %s into monitor list complete", insName)
}

func (mon *MonitorManager) Unregister(insName string) {
	glog.Infof("unregister mongo instance %s from monitor list", insName)
	mon.leave <- insName
	glog.Infof("unregister mongo instance %s from monitor list complete", insName)
}

//Go_Run is used for register mongo into monitor list
func (mon *MonitorManager) Go_Run() {
	for {
		glog.Infof("MonitorManager's Registry is running")
		select {
		case j := <-mon.join:
			glog.Infof("before insList: %v", mon.insList)
			if len(mon.insList) == 0 {
				mon.insList = append(mon.insList, j)
				glog.Infof("after insList: %v", mon.insList)
				break
			}
			for _, ins := range mon.insList {
				if ins == j {
					glog.Infof("after insList: %v", mon.insList)
					break
				}
			}
			glog.Infof("after insList: %v", mon.insList)
			mon.insList = append(mon.insList, j)
		case l := <-mon.leave:
			glog.Infof("before insList: %v", mon.insList)
			if len(l) == 0 {
				glog.Infof("after insList: %v", mon.insList)
				break
			}
			for index, ins := range mon.insList {
				if ins == l {
					mon.insList = append(mon.insList[:index-1], mon.insList[index+1:]...)
					glog.Infof("after insList: %v", mon.insList)
					break
				}
			}
			glog.Infof("after insList: %v", mon.insList)
		}
	}
}

//Add mongo instance into monitor list
func (mon *MonitorManager) Init() {
	for _, ins := range mon.ma.mongoMap {
		mon.insList = append(mon.insList, ins.Name)
	}
}
