package main

import (
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"time"
)

type Deployment interface {
	createMongo(*Mongo) error
	startMongo(*Mongo) error
	stopMongo(*Mongo) error
	deleteMongo(*Mongo) error
}

type MongoConf struct {
	Name     string
	Env      []string
	BasePath string
	DataPath string
	Version  string
}

const (
	DOCKER = iota
	NATIVE
)

type DeployType int

func NewDeployment(mm *MongoManager, d DeployType) Deployment {
	switch d {
	case DOCKER:
		return NewDockerDeployment(mm)
	case NATIVE:
		return NewNativeDeploymenet(mm)
	default:
		return NewDockerDeployment(mm)
	}
}

func getMongoConfFromMongoInstance(ins *Mongo) *MongoConf {
	conf := MongoConf{Name: ins.Name, BasePath: ins.BasePath, DataPath: ins.DataPath, Version: ins.Version}
	return &conf
}

func shutdownMongo(ins *Mongo, force bool) error {
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
