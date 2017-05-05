package main

import (
	"bytes"
	"errors"
	"fmt"
	cfgTmpl "github.com/GBjuno/dbagent/template"
	"github.com/golang/glog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	SingleDB = iota
	ReplsetDB
)

//MongoAgent is used for manage the mongodb db lifecycle.
type MongoAgent struct {
	insMap     map[string]MongoInstance
	statusMap  map[string]MongoInstanceStatus
	mongoMgr   MongoManager
	monitorMgr MonitorManager
}

func (ma *MongoAgent) WatchAndNotify() {

}

func (ma *MongoAgent) Send(changed interface{}) {
	switch changed.(type) {
	case *MongoInstance:
		glog.Infof("Publish /api/v1/MongoInstance/%s", changed.(MongoInstance).Name)
	case *MongoInstanceStatus:
		glog.Infof("Publish /api/v1/MongoInstance/%s/status", changed.(MongoInstanceStatus).Name)
	}
}

func (ma *MongoAgent) Init() {
	return ma
}

var insMongoAgent *MongoAgent
var once sync.Once

//NewMongoAgent is a *MongoAgent Singleton factory
func NewMongoAgent() MongoAgent {
	Duration(time.Now(), "NewMongoAgent")
	once.Do(func() {
		mongoMgr := MongoManager{insMongoAgent}
		monitorMgr := MonitorManager{insMongoAgent}
		insMongoAgent = MongoAgent{mongoMgr: mongoMgr, monitorMgr: monitorMgr}
	})
	return insMongoAgent
}
