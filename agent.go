package main

import (
	//"bytes"
	//"errors"
	"flag"
	//"fmt"
	"github.com/golang/glog"
	"net"
	"strings"
	"sync"
	"teego/pkg/api"
	"teego/pkg/client"
	"time"
)

//MongoAgent is used for manage the mongodb db lifecycle.
type MongoAgent struct {
	mongoMap   map[string]*api.MongoInstance
	mapLock    map[string]*sync.Mutex
	mongoMgr   *MongoManager
	monitorMgr *MonitorManager
	teegoCli   *client.Client
	lastResVer string
	node       string
}

func (ma *MongoAgent) GO_UpdateMongoInstance(ins *api.MongoInstance) {
	code, err := ma.teegoCli.MongoInstances(api.NamespaceAll).Update(ins.GetName(), ins)
	if err != nil {
		glog.Errorf("cannot update mongo instance %s", ins.GetName())
	}
	glog.Infof("update mongo instance %v succeed, return code", ins, code)
}

func (ma *MongoAgent) WatchAndNotify() {
	defer Duration(time.Now(), "WatchAndNotify")
	opts := &api.Options{}
	opts.ResourceVersion = ma.lastResVer
	watchInter, code, err := ma.teegoCli.MongoInstances(api.NamespaceAll).Watch(opts)
	if err != nil {
		glog.Errorf("watch return %d, %s", code, err)
	}
	for event := range watchInter.ResultChan() {
		glog.Infof("watch event: %#v", event)
		ins := event.Object.(*api.MongoInstance)
		if ins.Labels["node"] == ma.node {
			if _, ok := ma.mongoMap[ins.GetName()]; !ok {
				ma.mapLock[ins.GetName()] = &sync.Mutex{}
			}
			ma.mongoMap[ins.GetName()] = ins
			ma.mongoMap[ins.GetName()].Status.BasePath = "/opt/data"
			go ma.mongoMgr.GO_Handle(ma.mongoMap[ins.GetName()])
		}
	}
}

func (ma *MongoAgent) Init() error {
	opts := &api.Options{}
	result, code, err := ma.teegoCli.MongoInstances(api.NamespaceDefault).List(opts)
	if err != nil {
		glog.Errorf("list return %d, %s", code, err)
		return err
	}
	glog.Infof("list mongo instance: %#v", *result)
	ma.lastResVer = result.GetResourceVersion()
	for _, ins := range result.Items {
		glog.Infof("%s == %s", ins.Labels["node"], ma.node)
		if ins.Labels["node"] == ma.node {
			glog.Infof("instance match: %v", ins.GetName())
			ma.mongoMap[ins.GetName()] = &ins
			ma.mapLock[ins.GetName()] = &sync.Mutex{}
			ma.mongoMap[ins.GetName()].Status.BasePath = "/opt/data"
			oldStatus := ins.Status.Status
			ma.monitorMgr.simpleCheckOneIns(ins.GetName())
			newStatus := ins.Status.Status
			glog.Infof("instance %s, oldStatus %s, newStatus %s", ins.GetName(), oldStatus, newStatus)
			switch oldStatus {
			case CREATING:
				if ins.Status.DataPath == "" {
					if newStatus == RUNNING { //remove old instance not in registry but running and start a new one
						ins.Status.Status = ERROR
						ins.Status.Message = "an instance is aleady running but not in registry"
						go ma.GO_UpdateMongoInstance(&ins)
					} else if newStatus == STOPPED { //create a new instance
						ins.Status.Status = CREATING
						go ma.mongoMgr.GO_Handle(&ins)
					}
				} else {
					ins.Status.Status = ERROR
					ins.Status.Message = "mongodb has already been created"
					go ma.GO_UpdateMongoInstance(&ins)
				}
			case STARTING:
				if newStatus == RUNNING {
					go ma.GO_UpdateMongoInstance(&ins)
				} else if newStatus == STOPPED {
					ins.Status.Status = STARTING
					go ma.mongoMgr.GO_Handle(&ins)
				}
			case STOPPING:
				if newStatus == RUNNING {
					ins.Status.Status = STARTING
					go ma.mongoMgr.GO_Handle(&ins)
				} else if newStatus == STOPPED {
					go ma.GO_UpdateMongoInstance(&ins)
				}
			case RUNNING:
				if newStatus == RUNNING {
					go ma.GO_UpdateMongoInstance(&ins)
				} else if newStatus == STOPPED {
					ins.Status.Status = STARTING
					go ma.mongoMgr.GO_Handle(&ins)
				}
			case DELETING:
				ins.Status.Status = DELETING
				go ma.mongoMgr.GO_Handle(&ins)
			default:
				glog.Infof("no ops for current instance %s", ins.GetName())
			}
		}
	}
	ma.monitorMgr.Init()
	ma.WatchAndNotify()
	return nil
}

var mongoAgent *MongoAgent
var once sync.Once

//NewMongoAgent is a *MongoAgent Singleton factory
func NewMongoAgent() *MongoAgent {
	defer Duration(time.Now(), "NewMongoAgent")
	once.Do(func() {
		mongoMgr := NewMongoManager(NATIVE)
		monitorMgr := MonitorManager{insList: make([]string, 0), join: make(chan string), leave: make(chan string)}
		mongoMap := make(map[string]*api.MongoInstance)
		mapLock := make(map[string]*sync.Mutex)
		teegoCli := client.NewClient(apiServer)
		mongoAgent = &MongoAgent{mongoMap: mongoMap, mapLock: mapLock, mongoMgr: mongoMgr, monitorMgr: &monitorMgr, teegoCli: teegoCli, node: getLocalIP()}
		mongoMgr.ma = mongoAgent
		monitorMgr.ma = mongoAgent
		go monitorMgr.Go_Run()
		glog.Infof("mongoAgent create success %v", mongoAgent)
	})
	return mongoAgent
}

func getLocalIP() string {
	conn, err := net.Dial("tcp", "10.2.86.104:80")
	if err != nil {
		glog.Fatalf("cannot connect mongodb binary repository, err: %v", err)
	}
	defer conn.Close()
	ipaddr := strings.Split(conn.LocalAddr().String(), ":")[0]
	return ipaddr
}

var apiServer string

func main() {
	flag.StringVar(&apiServer, "apiserver", "http://127.0.0.1:8080", "api server address")
	flag.Parse()
	defer Duration(time.Now(), "main")
	mongoAgent := NewMongoAgent()
	mongoAgent.Init()
	return
}
