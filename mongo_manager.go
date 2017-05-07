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
	glog.Infof("flash mongo struct %s, currentOp: %s", ins.Name, ins.CurrOp)
	return nil
}

func (mm *MongoManager) Recovery() error {
	var oldOp string
	var dirList []string
	var basePathList []string
	for _, ins := range mm.ma.mongoMap {
		switch ins.CurrOp {
		case CREATE:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = START
			mm.GO_Handle(ins)
			ins.NextOp = oldOp
		case START:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = START
			if !ins.Running {
				mm.GO_Handle(ins)
			}
			ins.NextOp = oldOp
		case STOP:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = STOP
			if ins.Running {
				mm.GO_Handle(ins)
			}
			ins.NextOp = oldOp
		case DELETE:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = DELETE
			mm.GO_Handle(ins)
			ins.NextOp = oldOp
		case NOP:
		}
		dirList = append(dirList, ins.DataPath)
		if len(basePathList) == 0 {
			basePathList = append(basePathList, ins.BasePath)
		} else {
			for _, basePath := range basePathList {
				if ins.BasePath != basePath {
					basePathList = append(basePathList, ins.BasePath)
					break
				}
			}
		}
	}
	mm.CleanDir(dirList, basePathList)
	return nil
}

func (mm *MongoManager) CleanDir(dirList []string, basePathList []string) {
}

func (mm *MongoManager) GO_Handle(ins *Mongo) error {
	Duration(time.Now(), "NewMongoAgent")
	glog.Infof("MongoManager GO_Handle, mongo instance: %s, nextOp: %s", ins.Name, ins.NextOp)
	var err error
	switch ins.NextOp {
	case CREATE:
		if !ins.Created {
			ins.CurrOp = CREATE
			glog.Infof("creating instance %s", ins.Name)
			mm.Send(ins)
			if err = mm.createMongo(ins); err != nil {
				ins.Created = false
				ins.PrevOp = CREATE
				ins.CurrOp = ""
				ins.ValidOp = true
				mm.Send(ins)
				glog.Errorf("instance %s created failed", ins.Name)
				return err
			} else {
				ins.Created = true
				ins.PrevOp = CREATE
				ins.CurrOp = ""
				ins.ValidOp = true
				mm.Send(ins)
				glog.Infof("instance %s created success", ins.Name)
				mm.ma.monitorMgr.Register(ins.Name)
				return nil
			}
		} else {
			glog.Errorf("invalid operation %s on instance %s ", ins.NextOp, ins.Name)
			glog.Errorf("current operation %s on instance %s ", ins.CurrOp, ins.Name)
			glog.Errorf("instance %v ", ins)
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	case START:
		if !(ins.Running || ins.Deleted) && ins.CurrOp == "" {
			ins.CurrOp = START
			glog.Infof("start instance %s, containerID %s", ins.Name, ins.ContainerID)
			mm.Send(ins)
			for i := 0; i < MAXTRY; i++ {
				if err = mm.startMongo(ins); err == nil {
					break
				}
			}
			ins.PrevOp = START
			ins.CurrOp = ""
			ins.ValidOp = true
			mm.Send(ins)
			glog.Errorf("instance %v ", ins)
			return nil
		} else {
			glog.Errorf("invalid operation %s on instance %s ", ins.NextOp, ins.Name)
			glog.Errorf("current operation %s on instance %s ", ins.CurrOp, ins.Name)
			glog.Errorf("instance %v ", ins)
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	case STOP:
		glog.Errorf("instance %v ", ins)
		if ins.Running && !ins.Deleted && ins.CurrOp == "" {
			ins.CurrOp = STOP
			mm.Send(ins)
			for i := 0; i < MAXTRY; i++ {
				if err = mm.stopMongo(ins, false); err == nil {
					break
				}
			}
			ins.PrevOp = STOP
			ins.CurrOp = ""
			ins.ValidOp = true
			mm.Send(ins)
			return nil
		} else {
			glog.Errorf("invalid operation %s on instance %s ", ins.NextOp, ins.Name)
			glog.Errorf("current operation %s on instance %s ", ins.CurrOp, ins.Name)
			glog.Errorf("instance %v ", ins)
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	case DELETE:
		if ins.Created && ins.CurrOp == "" {
			ins.CurrOp = DELETE
			mm.Send(ins)
			if ins.Running {
				mm.stopMongo(ins, true)
			}
			for i := 0; i < MAXTRY; i++ {
				if err = mm.deleteMongo(ins); err == nil {
					break
				}
			}
			ins.Deleted = true
			ins.PrevOp = DELETE
			ins.CurrOp = ""
			ins.ValidOp = true
			mm.Send(ins)
			mm.ma.monitorMgr.Unregister(ins.Name)
			return nil
		} else {
			glog.Errorf("invalid operation %s on instance %s ", ins.NextOp, ins.Name)
			glog.Errorf("current operation %s on instance %s ", ins.CurrOp, ins.Name)
			glog.Errorf("instance %v ", ins)
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	default:
		return OpErr
	}
}

//createMongo is used for deploy mongo instance based on the configuration m
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

	//startup the mongodb instance by using docker
	resp, err := mm.dockerMgr.createContainer(ins)
	if err != nil {
		glog.Errorf("run docker container failed: %s", err.Error())
		goto RECOVER
	}

	ins.ContainerID = resp.ID
	mm.Send(ins)
	glog.Infof("create docker container success, id: %s", ins.ContainerID)
	return nil

RECOVER:
	os.RemoveAll(dataPath)
	glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", ins.Name)
	glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")

	return DeployErr
}

func (mm *MongoManager) startMongo(ins *Mongo) error {
	defer Duration(time.Now(), "startMongo")

	glog.Infof("starting mongo %s, container id %s", ins.Name, ins.ContainerID)
	if err := mm.dockerMgr.startContainer(ins.ContainerID); err != nil {
		glog.Errorf("start mongo %s and container id %s failed", ins.Name, ins.ContainerID)
		return err
	}

	glog.Infof("starting mongo %s, container id %s success", ins.Name, ins.ContainerID)
	go mm.ma.monitorMgr.simpleCheckOneIns(ins.Name)
	return nil
}

func (mm *MongoManager) stopMongo(ins *Mongo, force bool) error {
	defer Duration(time.Now(), "stopMongo")
	if err := mm.shutdownMongo(ins, force); err != nil {
		glog.Infof("shutdown mongo %s, container id %s", ins.Name, ins.ContainerID)
		return err
	}
	/*
		glog.Infof("stopping mongo %s, container id %s", ins.Name, ins.ContainerID)
		if err := dockerMgr.stopContainer(ins.ContainerID); err != nil {
			glog.Errorf("stop mongo %s and container id %s failed", ins.Name, ins.ContainerID)
			return err
		}
	*/
	glog.Infof("shutdown mongo %s, container id %s success", ins.Name, ins.ContainerID)
	go mm.ma.monitorMgr.simpleCheckOneIns(ins.Name)
	return nil
}

func (mm *MongoManager) deleteMongo(ins *Mongo) error {
	defer Duration(time.Now(), "stopMongo")

	glog.Infof("stopping mongo %s, container id %s", ins.Name, ins.ContainerID)
	if err := mm.dockerMgr.stopContainer(ins.ContainerID); err != nil {
		glog.Errorf("stop mongo %s and container id %s failed", ins.Name, ins.ContainerID)
		return err
	}

	glog.Infof("stop mongo %s, container id %s success", ins.Name, ins.ContainerID)
	return nil

}

func (mm *MongoManager) shutdownMongo(ins *Mongo, force bool) error {
	port := ins.Port
	session, err := mgo.DialWithTimeout(fmt.Sprintf("mongodb://127.0.0.1:%d/admin", port), time.Duration(5)*time.Second)
	if err != nil {
		glog.Errorf("connect to mongodb %s failed, port %d, error: %v", ins.Name, port, err)
		return err
	}
	glog.Infof("connect to mongodb %s succeed, port %d", ins.Name, port)
	defer session.Close()

	var result bson.M
	err = session.DB("admin").Run(bson.D{{"shutdown", 1}, {"force", force}}, &result)
	if err != nil {
		if err == io.EOF {
			glog.Infof("send shutdown mongodb succeed, port %d", port)
			glog.Infof("disconnect from mongodb %s after sending shutdown command", ins.Name)
			return nil
		}
		glog.Errorf("shutdown mongodb failed, port %d, error: %v, result: %v", port, err, result)
		return err
	}
	return nil
}
