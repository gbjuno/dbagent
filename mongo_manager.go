package main

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"teego/pkg/api"
	"time"
)

const (
	MAXTRY     = 3
	MAXTIMEOUT = 10
)

var DeployErr error = errors.New("Mongo Deploy Failed")
var OpErr error = errors.New("Invalid Operation on Mongo in current state")

type MongoManager struct {
	ma     *MongoAgent
	deploy Deployment
}

func NewMongoManager(d DeployType) *MongoManager {
	defer Duration(time.Now(), "NewMongoManager")
	glog.Infof("Start docker manager with request timeout=%v", MAXTIMEOUT)
	mongoMgr := &MongoManager{}
	mongoMgr.deploy = NewDeployment(mongoMgr, d)
	return mongoMgr
}

func (mm *MongoManager) GO_Handle(ins *api.MongoInstance) error {
	Duration(time.Now(), "GO_Handler")
	glog.Infof("MongoManager GO_Handle, mongo instance: %s, Status: %s", ins.GetName(), ins.Status.Status)
	var err error
	mm.ma.mapLock[ins.GetName()].Lock()
	defer mm.ma.mapLock[ins.GetName()].Unlock()
	switch ins.Status.Status {
	case CREATING:
		glog.Infof("creating mongo instance %s", ins.GetName())
		if err = mm.createMongo(ins); err != nil {
			ins.Status.Status = ERROR
			ins.Status.Message = err.Error()
			go mm.ma.GO_UpdateMongoInstance(ins)
			glog.Errorf("mongo instance %s created failed", ins.GetName())
			return err
		}
		if err = mm.startMongo(ins); err != nil {
			ins.Status.Status = ERROR
			ins.Status.Message = err.Error()
			go mm.ma.GO_UpdateMongoInstance(ins)
			glog.Errorf("mongo instance %s started failed", ins.GetName())
			return err
		}
		ins.Status.Status = RUNNING
		ins.Status.Message = fmt.Sprintf("mongo instance %s has been created and is running", ins.GetName())
		go mm.ma.GO_UpdateMongoInstance(ins)
		glog.Infof("create mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)
		mm.ma.monitorMgr.Register(ins.GetName())
		return nil
	case STARTING:
		glog.Infof("starting mongo instance %s, container id %s", ins.GetName(), ins.Status.Pid)
		for i := 0; i < MAXTRY; i++ {
			if err = mm.startMongo(ins); err == nil {
				ins.Status.Status = RUNNING
				ins.Status.Message = fmt.Sprintf("mongo instance %s has been started and is running", ins.GetName())
				go mm.ma.GO_UpdateMongoInstance(ins)
				glog.Infof("start mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)
				return nil
			}
		}
		ins.Status.Status = ERROR
		ins.Status.Message = err.Error()
		go mm.ma.GO_UpdateMongoInstance(ins)
		glog.Errorf("start mongo instance %s, container id %s failed", ins.GetName(), ins.Status.Pid)
		return err
	case STOPPING:
		glog.Infof("stopping mongo instance %s, container id %s", ins.GetName(), ins.Status.Pid)
		for i := 0; i < MAXTRY; i++ {
			if err = mm.stopMongo(ins); err == nil {
				ins.Status.Status = STOPPED
				ins.Status.Message = fmt.Sprintf("mongo instance %s has been stopped and is not running", ins.GetName())
				go mm.ma.GO_UpdateMongoInstance(ins)
				glog.Infof("stop mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)
				return nil
			}
		}
		ins.Status.Status = ERROR
		ins.Status.Message = err.Error()
		go mm.ma.GO_UpdateMongoInstance(ins)
		glog.Errorf("stop mongo instance %s, container id %s failed", ins.GetName(), ins.Status.Pid)
		return err
	case DELETING:
		if ins.Status.Status == RUNNING {
			mm.stopMongo(ins)
		}
		for i := 0; i < MAXTRY; i++ {
			if err = mm.deleteMongo(ins); err == nil {
				ins.Status.Status = DELETED
				ins.Status.Message = fmt.Sprintf("mongo instance %s has been deleted", ins.GetName())
				go mm.ma.GO_UpdateMongoInstance(ins)
				mm.ma.monitorMgr.Unregister(ins.GetName())
				glog.Infof("Delete mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)
				return nil
			}
		}
		ins.Status.Status = ERROR
		ins.Status.Message = err.Error()
		go mm.ma.GO_UpdateMongoInstance(ins)
		glog.Errorf("Delete mongo instance %s, container id %s failed", ins.GetName(), ins.Status.Pid)
		return nil
	default:
		return nil
	}
}

//createMongo is used for createMongo
func (mm *MongoManager) createMongo(ins *api.MongoInstance) error {
	return mm.deploy.createMongo(ins)
}

//startMongo is used for start mongo
func (mm *MongoManager) startMongo(ins *api.MongoInstance) error {
	return mm.deploy.startMongo(ins)
}

//stopMongo is used for stop mongo
func (mm *MongoManager) stopMongo(ins *api.MongoInstance) error {
	return mm.deploy.stopMongo(ins)
}

//deleteMongo is used for delete mongo
func (mm *MongoManager) deleteMongo(ins *api.MongoInstance) error {
	return mm.deploy.deleteMongo(ins)
}
