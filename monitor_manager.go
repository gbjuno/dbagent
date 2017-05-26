package main

import (
	//	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	//"reflect"
	"teego/pkg/api"
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
			go mon.ma.GO_UpdateMongoInstance(mon.ma.mongoMap[insName])
		}
		time.Sleep(60 * time.Second)
	}
}

func (mon *MonitorManager) simpleCheckOneIns(insName string) {
	mon.ma.mapLock[insName].Lock()
	defer mon.ma.mapLock[insName].Unlock()
	ins := mon.ma.mongoMap[insName]
	glog.Infof("checking mongo instance %s status...", ins.GetName())
	conn, err := mgo.Dial(fmt.Sprintf("127.0.0.1:%d/admin", ins.Spec.Port))
	if err != nil {
		glog.Errorf("connect mongo instance %s failed", ins.GetName())
		glog.Errorf("mongo instance %s is not running", ins.GetName())
		ins.Status.Message = "Monitor: mongodb is not running"
		ins.Status.Status = STOPPED
		return
	}
	defer conn.Close()
	c := conn.DB("local").C("startup_log")

	var result interface{}
	err = c.Find(nil).One(&result)
	if err != nil {
		glog.Errorf("cannot get data of DB local Table startup_log of mongo %s", ins.GetName())
		glog.Errorf("mongo instance %s is not running property", ins.GetName())
		ins.Status.Message = "Monitor: running but without startup_log"
		ins.Status.Status = ERROR
		return
	}
	glog.Infof("mongo instance %s is alive and ready", ins.GetName())
	ins.Status.Status = RUNNING
	ins.Status.Message = "Monitor: mongodb is running"
	glog.Infof("instance %v", ins)
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

func (mon *MonitorManager) getMongoStatus(ins *api.MongoInstance) {
	mon.ma.mapLock[ins.GetName()].Lock()
	defer mon.ma.mapLock[ins.GetName()].Unlock()
	conn, err := mgo.Dial(fmt.Sprintf("127.0.0.1:%d/admin", ins.Spec.Port))
	if err != nil {
		glog.Errorf("connect mongo instance %s failed", ins.GetName())
		glog.Errorf("mongo instance %s is not running", ins.GetName())
		ins.Status.Status = STOPPED
		ins.Status.Message = "mongodb is not running"
		return
	}
	var result bson.M
	err = conn.DB("admin").Run(bson.D{{"serverStatus", 1}}, &result)
	if err != nil {
		glog.Errorf("get Mongo Server Status failed, err %c", err)
		return
	}
	defer conn.Close()
	/*
		serverStatus, err := json.Marshal(result)
		if err != nil {
			glog.Errorf("get Mongo Server Status failed, err %c", err)
			return
		}
	*/

	asserts := Asserts{}
	network := Network{}
	connections := Connections{}
	//dur := Dur{}
	opcounters := Opcounters{}
	storageEngine := StorageEngine{}
	mem := Mem{}

	asserts.Regular = result["asserts"].(bson.M)["regular"].(int)
	asserts.Warning = result["asserts"].(bson.M)["warning"].(int)
	asserts.Msg = result["asserts"].(bson.M)["msg"].(int)
	asserts.User = result["asserts"].(bson.M)["user"].(int)
	asserts.Msg = result["asserts"].(bson.M)["rollovers"].(int)

	network.BytesIn = result["network"].(bson.M)["bytesIn"].(int64)
	network.BytesOut = result["network"].(bson.M)["bytesOut"].(int64)
	network.NumRequests = result["network"].(bson.M)["numRequests"].(int64)

	connections.Available = result["connections"].(bson.M)["available"].(int)
	connections.Current = result["connections"].(bson.M)["current"].(int)
	connections.TotalCreated = result["connections"].(bson.M)["totalCreated"].(int)
	/*
		dur.Commits = result["dur"].(bson.M)["commits"].(int64)
		dur.CommitsInWriteLock = result["dur"].(bson.M)["commitsInWriteLock"].(int64)
		dur.Compression = result["dur"].(bson.M)["compression"].(int64)
		dur.EarlyCommits = result["dur"].(bson.M)["earlyCommits"].(int64)
		dur.JournaledMB = result["dur"].(bson.M)["journaledMB"].(int64)
		dur.WriteToDataFilesMB = result["dur"].(bson.M)["writeToDataFilesMB"].(int64)

		dur.DTimeMs.Commits = result["dur"].(bson.M)["timeMs"].(bson.M)["commits"].(int64)
		dur.DTimeMs.CommitsInWriteLock = result["dur"].(bson.M)["timeMs"].(bson.M)["commitsInWriteLock"].(int64)
		dur.DTimeMs.Dt = result["dur"].(bson.M)["timeMs"].(bson.M)["dt"].(int64)
		dur.DTimeMs.PrepLogBuffer = result["dur"].(bson.M)["timeMs"].(bson.M)["prepLogBuffer"].(int64)
		dur.DTimeMs.RemapPrivateView = result["dur"].(bson.M)["timeMs"].(bson.M)["remapPrivateView"].(int64)
		dur.DTimeMs.WriteToDataFiles = result["dur"].(bson.M)["timeMs"].(bson.M)["writeToDataFiles"].(int64)
		dur.DTimeMs.WriteToJournal = result["dur"].(bson.M)["timeMs"].(bson.M)["writeToJournal"].(int64)
	*/
	opcounters.Command = result["opcounters"].(bson.M)["command"].(int)
	opcounters.Delete = result["opcounters"].(bson.M)["delete"].(int)
	opcounters.Getmore = result["opcounters"].(bson.M)["getmore"].(int)
	opcounters.Insert = result["opcounters"].(bson.M)["insert"].(int)
	opcounters.Query = result["opcounters"].(bson.M)["query"].(int)
	opcounters.Update = result["opcounters"].(bson.M)["update"].(int)

	storageEngine.Name = result["storageEngine"].(bson.M)["name"].(string)
	storageEngine.Persistent = result["storageEngine"].(bson.M)["persistent"].(bool)
	storageEngine.SupportsCommittedReads = result["storageEngine"].(bson.M)["supportsCommittedReads"].(bool)

	mem.Bits = result["mem"].(bson.M)["bits"].(int)
	mem.Mapped = result["mem"].(bson.M)["mapped"].(int)
	mem.MappedWithJournal = result["mem"].(bson.M)["mappedWithJournal"].(int)
	mem.Resident = result["mem"].(bson.M)["resident"].(int)
	mem.Supported = result["mem"].(bson.M)["supported"].(bool)
	mem.Virtual = result["mem"].(bson.M)["virtual"].(int)

	glog.Infof("%v", asserts)
	glog.Infof("%v", network)
	glog.Infof("%v", connections)
	glog.Infof("%v", opcounters)
	glog.Infof("%v", storageEngine)

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
			mon.insList = append(mon.insList, j)
			glog.Infof("after insList: %v", mon.insList)
		case l := <-mon.leave:
			glog.Infof("before insList: %v", mon.insList)
			if len(l) == 0 {
				glog.Infof("after insList: %v", mon.insList)
				break
			}
			for index, ins := range mon.insList {
				if ins == l {
					mon.insList = append(mon.insList[:index], mon.insList[index+1:]...)
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
		if ins.Status.Status != DELETED || ins.Status.Status != CREATING {
			mon.insList = append(mon.insList, ins.GetName())
		}
	}
	glog.Infof("monitorManager is initilized, monitor List: %v", mon.insList)
	go mon.Go_Run()
	go mon.monitorAll()
}
