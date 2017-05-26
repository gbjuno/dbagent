package main

import (
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"teego/pkg/api"
	"time"
)

type Deployment interface {
	createMongo(*api.MongoInstance) error
	startMongo(*api.MongoInstance) error
	stopMongo(*api.MongoInstance) error
	deleteMongo(*api.MongoInstance) error
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

func getMongoConfFromMongoInstance(ins *api.MongoInstance) *MongoConf {
	conf := MongoConf{Name: ins.GetName(), BasePath: ins.Status.BasePath, DataPath: ins.Status.DataPath, Version: ins.Spec.Version}
	return &conf
}

func shutdownMongo(ins *api.MongoInstance, force bool) error {
	defer Duration(time.Now(), "shutdownMongo")
	port := ins.Spec.Port
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
