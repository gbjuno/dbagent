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

var DeployErr error = errors.New("Mongo Deploy Failed")

type Event interface{}

type Notifier interface {
	Notify()
	Register(*Observer)
	Unregister(*Observer)
}

type Observer interface {
	Notify(Event)
}

//EventNotifier watch APIServer, get deployment configuration
//and notify all registered observers to act.
type EventNotifier struct {
	obPtrList []*Observer
}

func (en *EventNotifier) Register(ob *Observer) {
	for _, v := range en.obPtrList {
		if ob == v {
			return
		}
	}
	en.obPtrList = append(en.obPtrList, ob)
}

func (en *EventNotifier) Unregister(ob *Observer) {
	for k, v := range en.obPtrList {
		if ob == v {
			en.obPtrList = append(en.obPtrList[:k], en.obPtrList[k+1:]...)
		}
	}
}

func (en *EventNotifier) Notify() {
	for _, v := range en.obPtrList {
		fmt.Println("v", v)
	}
}

func (en *EventNotifier) Watch() {
}

func (en *EventNotifier) Run() {
	fmt.Println("event notifier is running")
}

//EventObserver get notified by EventNotifier and then act according to the configurations.
type EventObserver struct {
}

func (eo *EventObserver) Notify(e Event) {
	fmt.Printf("%v\n", e)
}

//AutoAgent is used for deployment of mongodbs.
//AutoAgent has an EventNotifier to watch APIServer to get deployment configurations.
type AutoAgent struct {
	en *EventNotifier
}

type MongoInstance struct {
	Name     string
	Port     int
	BasePath string
	DataPath string
	Version  string
}

func (a *AutoAgent) Notify(e Event) {
}

func (a *AutoAgent) Run() {
	go a.en.Run()
}

//deployMongoIns is used for deploy mongo instance based on the configuration m
func (a *AutoAgent) deployMongoIns(m *MongoInstance) error {
	Duration(time.Now(), "deployMongoIns")

	var err error
	var f *os.File
	var tmpl *template.Template
	var conParam []string
	var cmd *exec.Cmd
	var cmdStdout bytes.Buffer
	var cmdStderr bytes.Buffer
	var now time.Time = time.Now()
	var dataPath string = fmt.Sprintf("%s/%s_%04d%02d%02d_%02d%02d", m.BasePath, m.Name,
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())

	if _, err := os.Stat(m.BasePath); os.IsNotExist(err) {
		glog.Errorf("BasePath %s does not exist", m.BasePath)
		return DeployErr
	}

	//create mongo datapath
	if err = os.Mkdir(dataPath, os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath)
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath, m.Name)

	m.DataPath = dataPath

	//create configuration file
	glog.Infof("creating configuration file mongodb.conf for mongo %s", m.Name)
	f, err = os.OpenFile(dataPath+"/mongodb.conf", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	if err != nil {
		glog.Errorf("can not create configuration file %s", dataPath+"/mongodb.conf")
		goto RECOVER
	}

	tmpl, err = template.New("replset").Parse(cfgTmpl.Replset)
	if err != nil {
		glog.Errorf("can not template replset")
		goto RECOVER
	}

	err = tmpl.Execute(f, m)
	if err != nil {
		glog.Errorf("can not template replset")
		goto RECOVER
	}
	glog.Infof("create file %s for mongo %s", dataPath+"/mongodb.conf", m.Name)

	//startup the mongodb instance by using docker
	conParam = []string{"-H", "127.0.0.1:4321", "run", "-itd", "--network", "host",
		"--name", m.Name, "-v", m.DataPath + ":/data",
		fmt.Sprintf("docker.gf.com.cn/gf-mongodb:%s", m.Version),
		"bash", "-c", "/usr/bin/mongod -f /data/mongodb.conf"}

	cmd = exec.Command("docker", conParam...)
	cmd.Stdout = &cmdStdout
	cmd.Stderr = &cmdStderr
	err = cmd.Start()

	if err != nil {
		glog.Errorf("run docker container failed: %s", err.Error())
		goto RECOVER
	}

	glog.Infof("wait for docker container up")
	err = cmd.Wait()

	if err != nil {
		glog.Errorf("run docker container failed: %s, %s", err.Error(),
			strings.Replace(string(cmdStderr.Bytes()), "\n", " ", -1))
		goto RECOVER
	}

	glog.Infof("create docker container success, id: %s",
		strings.Replace(string(cmdStdout.Bytes()), "\n", " ", -1))

	return nil

RECOVER:
	os.RemoveAll(dataPath)
	glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", m.Name)
	glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
	return DeployErr
}

var insAutoAgent *AutoAgent
var once sync.Once

//NewAutoAgent is a *AutoAgent Singleton factory
func NewAutoAgent() *AutoAgent {
	Duration(time.Now(), "NewAutoAgent")
	once.Do(func() {
		var obPtrList []*Observer = make([]*Observer, 10)
		var en *EventNotifier = &EventNotifier{obPtrList}
		insAutoAgent = &AutoAgent{en}
	})
	return insAutoAgent
}
