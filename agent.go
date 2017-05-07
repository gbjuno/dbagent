package main

import (
	//"bytes"
	//"errors"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"sync"
	"time"
)

const (
	SingleDB = iota
	ReplsetDB
)

//MongoAgent is used for manage the mongodb db lifecycle.
type MongoAgent struct {
	mongoMap   map[string]*Mongo
	mongoMgr   *MongoManager
	monitorMgr *MonitorManager
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

var mongoAgent *MongoAgent
var once sync.Once

//NewMongoAgent is a *MongoAgent Singleton factory
func NewMongoAgent() *MongoAgent {
	defer Duration(time.Now(), "NewMongoAgent")
	once.Do(func() {
		mongoMgr := NewMongoManager()
		monitorMgr := MonitorManager{insList: make([]string, 0), join: make(chan string), leave: make(chan string)}
		mongoMap := make(map[string]*Mongo)
		mongoAgent = &MongoAgent{mongoMap: mongoMap, mongoMgr: mongoMgr, monitorMgr: &monitorMgr}
		mongoMgr.ma = mongoAgent
		monitorMgr.ma = mongoAgent
		go monitorMgr.Go_Run()
		glog.Infof("mongoAgent create success %v", mongoAgent)
	})
	glog.Infof("return mongoAgent")
	return mongoAgent
}

func main() {
	flag.Parse()
	defer Duration(time.Now(), "main")
	mongoAgent := NewMongoAgent()
	for i := 0; i < 5; i++ {
		mongoAgent.mongoMap[fmt.Sprintf("test%d", i)] = &Mongo{
			Name:        fmt.Sprintf("test%d", i),
			BasePath:    "/opt/data/",
			Role:        "SingleDB",
			Port:        27000 + i,
			CacheSizeMB: 10240,
			Version:     "3.2.11",
			Type:        SingleDB,
			NextOp:      "CREATE",
		}
		glog.Infof("create mongo struct %v", mongoAgent.mongoMap[fmt.Sprintf("test%d", i)])
	}
	glog.Infof("get mongoAgent success")
	ins := mongoAgent.mongoMap["test0"]
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		glog.Errorf("start a mongo instance failed")
	}
	return
}
