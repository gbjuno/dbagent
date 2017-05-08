package main

import (
	"errors"
	"fmt"
	cfgTmpl "github.com/GBjuno/dbagent/template"
	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"os"
	"text/template"
	"time"
)

const (
	MAXTRY     = 3
	MAXTIMEOUT = 10
)

var DeployErr error = errors.New("Mongo Deploy Failed")
var OpErr error = errors.New("Invalid Operation on Mongo in current state")

type MongoManager struct {
	ma        *MongoAgent
	dockerMgr *DockerManager
}

func NewMongoManager() *MongoManager {
	defer Duration(time.Now(), "NewMongoManager")
	dockerMgr := NewDockerManager()
	glog.Infof("Start docker manager with request timeout=%v", MAXTIMEOUT)
	mongoMgr := &MongoManager{dockerMgr: dockerMgr}
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
			if err = mm.stopMongo(ins, false); err == nil {
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
			mm.stopMongo(ins, true)
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

//createMongo is used for deploy mongo mongo instance based on the configuration m
func (mm *MongoManager) createMongo(ins *Mongo) error {
	defer Duration(time.Now(), "createMongo")

	var err error
	var f *os.File
	var tmpl *template.Template
	var tmplConf string
	var now time.Time = time.Now()
	var dataPath string = fmt.Sprintf("%s/%s_%04d%02d%02d_%02d%02d", ins.BasePath, ins.Name,
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())

	if _, err := os.Stat(ins.BasePath); os.IsNotExist(err) {
		glog.Errorf("BasePath %s does not exist", ins.BasePath)
		return DeployErr
	}

	//create mongo datapath
	if err = os.Mkdir(dataPath, os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath)
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath, ins.Name)

	ins.DataPath = dataPath

	//create configuration file
	glog.Infof("creating configuration file mongodb.conf for mongo %s", ins.Name)
	f, err = os.OpenFile(dataPath+"/mongodb.conf", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	if err != nil {
		glog.Errorf("can not create configuration file %s", dataPath+"/mongodb.conf")
		os.RemoveAll(dataPath)
		glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", ins.Name)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}

	switch ins.Type {
	case SingleDB:
		tmplConf = cfgTmpl.Single
	case ReplsetDB:
		tmplConf = cfgTmpl.Replset
	default:
		tmplConf = cfgTmpl.Single
	}

	tmpl, err = template.New("db").Parse(tmplConf)
	if err != nil {
		glog.Errorf("can not template %d", tmplConf)
		os.RemoveAll(dataPath)
		glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", ins.Name)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}

	err = tmpl.Execute(f, ins)
	if err != nil {
		glog.Errorf("can not template %d", tmplConf)
		os.RemoveAll(dataPath)
		glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", ins.Name)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}
	glog.Infof("create file %s for mongo %s", dataPath+"/mongodb.conf", ins.Name)

	//startup the mongodb mongo instance by using docker
	resp, err := mm.dockerMgr.createContainer(ins)
	if err != nil {
		glog.Errorf("run docker container failed: %s", err.Error())
		goto RECOVER
	}

	ins.ContainerID = resp.ID
	mm.Send(ins)
	glog.Infof("create docker container succeed, id: %s", ins.ContainerID)
	return nil

RECOVER:
	os.RemoveAll(dataPath)
	glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")

	return DeployErr
}

func (mm *MongoManager) startMongo(ins *Mongo) error {
	defer Duration(time.Now(), "startMongo")

	glog.Infof("starting mongo instance %s, container id %s", ins.Name, ins.ContainerID)
	if err := mm.dockerMgr.startContainer(ins.ContainerID); err != nil {
		glog.Errorf("start mongo instance %s and container id %s failed, err: %v", ins.Name, ins.ContainerID, err)
		return err
	}

	glog.Infof("start mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)
	go mm.ma.monitorMgr.simpleCheckOneIns(ins.Name)
	return nil
}

func (mm *MongoManager) stopMongo(ins *Mongo, force bool) error {
	defer Duration(time.Now(), "stopMongo")

	glog.Infof("stopping mongo instance %s, container id %s", ins.Name, ins.ContainerID)
	if err := mm.shutdownMongo(ins, force); err != nil {
		glog.Infof("stop mongo instance %s, container id %s failed, err: %v", ins.Name, ins.ContainerID, err)
		return err
	}
	/*
		glog.Infof("stopping mongo %s, container id %s", ins.Name, ins.ContainerID)
		if err := dockerMgr.stopContainer(ins.ContainerID); err != nil {
			glog.Errorf("stop mongo %s and container id %s failed", ins.Name, ins.ContainerID)
			return err
		}
	*/
	glog.Infof("stop mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)
	go mm.ma.monitorMgr.simpleCheckOneIns(ins.Name)
	return nil
}

func (mm *MongoManager) deleteMongo(ins *Mongo) error {
	defer Duration(time.Now(), "deleteMongo")

	glog.Infof("stopping mongo instance %s, container id %s", ins.Name, ins.ContainerID)
	if err := mm.dockerMgr.stopContainer(ins.ContainerID); err != nil {
		glog.Errorf("stop mongo instance %s and container id %s failed, err: %v", ins.Name, ins.ContainerID, err)
		return err
	}
	glog.Infof("stop mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)

	glog.Infof("removing mongo instance %s, container id %s", ins.Name, ins.ContainerID)
	if err := mm.dockerMgr.removeContainer(ins.ContainerID); err != nil {
		glog.Errorf("remove mongo %s and container id %s failed, err: %v", ins.Name, ins.ContainerID, err)
		return err
	}
	glog.Infof("remove mongo instance %s, container id %s succeed", ins.Name, ins.ContainerID)

	os.RemoveAll(ins.DataPath)
	glog.Infof("remove mongo instance directory %s", ins.DataPath)
	return nil

}

func (mm *MongoManager) shutdownMongo(ins *Mongo, force bool) error {
	defer Duration(time.Now(), "shutdownMongo")
	port := ins.Port
	session, err := mgo.DialWithTimeout(fmt.Sprintf("mongodb://127.0.0.1:%d/admin", port), time.Duration(5)*time.Second)
	if err != nil {
		glog.Errorf("connect to mongo instance %s failed, port %d, error: %v", ins.Name, port, err)
		return err
	}
	glog.Infof("connect to mongo instance %s succeed, port %d", ins.Name, port)
	defer session.Close()

	var result bson.M
	err = session.DB("admin").Run(bson.D{{"shutdown", 1}, {"force", force}}, &result)
	if err != nil {
		if err == io.EOF {
			glog.Infof("send shutdown mongo instance succeed, port %d", port)
			glog.Infof("disconnect from mongo instance %s after sending shutdown command", ins.Name)
			return nil
		}
		glog.Errorf("shutdown mongo instance failed, port %d, error: %v, result: %v", port, err, result)
		return err
	}
	return nil
}
