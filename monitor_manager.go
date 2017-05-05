package main

import (
	"errors"
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

func (mon *MonitorManager) simpleCheckOneIns(insName string) error {
	ins := mon.ma.insMap[insName]
	glog.infof("checking mongo instance %s status...", ins.Name)
	conn, err := mgo.Dial(fmt.Sprintf("127.0.0.1:%d/admin", ins.Port))
	if err != nil {
		glog.Errorf("connect mongo %s failed, port %d", ins.Name, ins.Port)
		return MonitorErr
	}
	defer conn.Close()

	c := conn.DB("local").C("me")

	var result interface{}
	err = c.Find().One(&result)
	if err != nil {
		glog.Errorf("cannot get data of DB local Table me")
	}
	glog.Infof("mong instance %s is alive and ready", ins.Name)

	return nil
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
		case j := <-join:
			for _, ins := range mon.insList {
				if ins == j {
					break
				}
			}
			mon.insList = append(mon.insList, j)
		case l := <-leave:
			for index, ins := range mon.insList {
				if ins == l {
					mon.insList = append(mon.insList[:index-1], mon.insList[index+1:])
				}
			}
			break
		}
	}
}

//Add mongo instance into monitor list
func (mon *MonitorManager) Init() error {
	for _, ins := range mon.ma.mongoMap {
		if ins.Created && !ins.Deleted {
			mon.insList = append(mon.insList, ins.Name)
		}
	}
}
