package main

import (
	"errors"
	"github.com/golang/glog"
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

func (mm *MongoManager) Send(ins *Mongo) error {
	glog.Infof("flash mongo struct %s, currentOp: %s", ins.Name, ins.Status)
	return nil
}

func (mm *MongoManager) GO_Handle(ins *Mongo) error {
	Duration(time.Now(), "GO_Handler")
	glog.Infof("MongoManager GO_Handle, mongo instance: %s, Status: %s", ins.Name, ins.Status)
	var err error
	ins.locker.Lock()
	defer ins.locker.Unlock()
	switch ins.Status {
	case CREATING:
		glog.Infof("creating mongo instance %s", ins.Name)
		if err = mm.createMongo(ins); err != nil {
			ins.Status = ERROR
			mm.Send(ins)
			glog.Errorf("mongo instance %s created failed", ins.Name)
			return err
		}
		if err = mm.startMongo(ins); err != nil {
			ins.Status = ERROR
			glog.Errorf("mongo instance %s started failed", ins.Name)
			return err
		}
		ins.Status = RUNNING
		mm.Send(ins)
		glog.Infof("create mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)
		mm.ma.monitorMgr.Register(ins.Name)
		return nil
	case STARTING:
		glog.Infof("starting mongo instance %s, container id %s", ins.Name, ins.ContainerID)
		for i := 0; i < MAXTRY; i++ {
			if err = mm.startMongo(ins); err == nil {
				ins.Status = RUNNING
				mm.Send(ins)
				glog.Infof("start mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)
				return nil
			}
		}
		ins.Status = ERROR
		mm.Send(ins)
		glog.Errorf("start mongo instance %s, container id %s failed", ins.Name, ins.ContainerID)
		return err
	case STOPPING:
		glog.Infof("stopping mongo instance %s, container id %s", ins.Name, ins.ContainerID)
		for i := 0; i < MAXTRY; i++ {
			if err = mm.stopMongo(ins); err == nil {
				ins.Status = STOPPED
				mm.Send(ins)
				glog.Infof("stop mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)
				return nil
			}
		}
		ins.Status = STOPPED
		mm.Send(ins)
		glog.Errorf("stop mongo instance %s, container id %s failed", ins.Name, ins.ContainerID)
		return err
	case DELETING:
		if ins.Running {
			mm.stopMongo(ins)
		}
		for i := 0; i < MAXTRY; i++ {
			if err = mm.deleteMongo(ins); err == nil {
				ins.Status = DELETED
				mm.Send(ins)
				glog.Infof("Delete mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)
				return nil
			}
		}
		ins.Status = ERROR
		mm.Send(ins)
		glog.Errorf("Delete mongo instance %s, container id %s failed", ins.Name, ins.ContainerID)
		mm.ma.monitorMgr.Unregister(ins.Name)
		return nil
	default:
		return nil
	}
}

//createMongo is used for createMongo
func (mm *MongoManager) createMongo(ins *Mongo) error {
	return mm.deploy.createMongo(ins)
}

//startMongo is used for start mongo
func (mm *MongoManager) startMongo(ins *Mongo) error {
	return mm.deploy.startMongo(ins)
}

//stopMongo is used for stop mongo
func (mm *MongoManager) stopMongo(ins *Mongo) error {
	return mm.deploy.stopMongo(ins)
}

//deleteMongo is used for delete mongo
func (mm *MongoManager) deleteMongo(ins *Mongo) error {
	return mm.deploy.deleteMongo(ins)
}
