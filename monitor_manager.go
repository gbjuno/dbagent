package main

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
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

func (mon *MonitorManager) simpleCheckOneIns(insName string) {
	ins := mon.ma.mongoMap[insName]
	glog.Infof("checking mongo instance %s status...", ins.Name)
	conn, err := mgo.Dial(fmt.Sprintf("127.0.0.1:%d/admin", ins.Port))
	defer conn.Close()
	if err != nil {
		glog.Errorf("connect mongo %s failed", ins.Name)
		glog.Errorf("mongo %s is not running", ins.Name)
		ins.Running = false
		mon.ma.mongoMgr.Send(&ins)
		return
	}
	c := conn.DB("local").C("me")

	var result interface{}
	err = c.Find("").One(&result)
	if err != nil {
		glog.Errorf("cannot get data of DB local Table me of mongo %s", ins.Name)
		ins.Running = false
		mon.ma.mongoMgr.Send(&ins)
		return
	}
	glog.Infof("mong instance %s is alive and ready", ins.Name)
	ins.Running = true
	mon.ma.mongoMgr.Send(&ins)
}

func (mon *MonitorManager) Register(insName string) {
	mon.join <- insName
}

func (mon *MonitorManager) Unregister(insName string) {
	mon.leave <- insName
}

//Go_Run is used for register mongo into monitor list
func (mon *MonitorManager) Go_Run() {
	for {
		select {
		case j := <-mon.join:
			for _, ins := range mon.insList {
				if ins == j {
					break
				}
			}
			mon.insList = append(mon.insList, j)
		case l := <-mon.leave:
			for index, ins := range mon.insList {
				if ins == l {
					mon.insList = append(mon.insList[:index-1], mon.insList[index+1:]...)
				}
			}
			break
		}
	}
}

//Add mongo instance into monitor list
func (mon *MonitorManager) Init() {
	for _, ins := range mon.ma.mongoMap {
		if ins.Created && !ins.Deleted {
			mon.insList = append(mon.insList, ins.Name)
		}
	}
}
