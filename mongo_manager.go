package main

import (
	"time"
)

var DeployErr error = errors.New("Mongo Deploy Failed")
var ParamErr error = errors.New("Invalid Parameter")
var OpErr error = errors.New("Invalid Operation on Mongo in current state")

type MongoManager struct {
	ma         *MongoAgent
	contextMap map[string]MongoContext
}

func (mm *MongoManager) GO_Handle(insName string) error {
	ins := mm.ma.insMap[insName]
	insStatus := mm.ma.statusMap[insName]

	if insStatus.LastStatus == "" && insStatus.Status == "" {
		if ins.NextOp == "CREATE" {
			mm.contextMap[insName] = MongoContext{mm: mm, ins: ins, insStatus: insStatus}
		} else {
			return OpErr
		}
	}

	if insState, ok := mm.contextMap[insName]; ok {
	} else {
		return OpErr
	}
}

type MongoContext struct {
	mm        *MongoManager
	ins       *MongoInstance
	insStatus *MongoInstanceStatus
}

func (mc *MongoContext) Run() {
	for {
		op := <-mc.operation
		fn[op](mc)
	}
}

//createMongo is used for deploy mongo instance based on the configuration m
func createMongo(mc *MongoContext) error {
	defer Duration(time.Now(), "createMongo")

	var insName = mc.ins.Name
	var ins MongoInstance = mc.ins
	var insStatus MongoInstanceStatus = mc.insStatus

	mc.insStatus.Status = "creating"
	mc.insStatus.LastUpdate = time.Now()
	mc.insStatus.CurrOp = "CREATE"
	mc.mm.ma.Send(&ins)

	var err error
	var f *os.File
	var tmpl *template.Template
	var tmplConf string
	var conParam []string
	var cmd *exec.Cmd
	var cmdStdout bytes.Buffer
	var cmdStderr bytes.Buffer
	var now time.Time = time.Now()
	var dataPath string = fmt.Sprintf("%s/%s_%04d%02d%02d_%02d%02d", ins.BasePath, ins.Name,
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())

	if _, err := os.Stat(ins.BasePath); os.IsNotExist(err) {
		glog.Errorf("BasePath %s does not exist", ins.BasePath)
		mc.insStatus.LastStatus = "creating"
		mc.insStatus.Status = "error"
		mc.insStatus.LastUpdate = time.Now()
		mc.mm.ma.Send(&ins)
		return DeployErr
	}

	//create mongo datapath
	if err = os.Mkdir(dataPath, os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath)
		mc.insStatus.LastStatus = "creating"
		mc.insStatus.Status = "error"
		mc.insStatus.LastUpdate = time.Now()
		mc.mm.ma.Send(&ins)
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath, ins.Name)

	insStatus.DataPath = dataPath

	//create configuration file
	glog.Infof("creating configuration file mongodb.conf for mongo %s", ins.Name)
	f, err = os.OpenFile(dataPath+"/mongodb.conf", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	if err != nil {
		glog.Errorf("can not create configuration file %s", dataPath+"/mongodb.conf")
		goto RECOVER
	}

	switch m.Type {
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
		goto RECOVER
	}

	err = tmpl.Execute(f, ins)
	if err != nil {
		glog.Errorf("can not template %d", tmplConf)
		goto RECOVER
	}
	glog.Infof("create file %s for mongo %s", dataPath+"/mongodb.conf", ins.Name)

	//startup the mongodb instance by using docker
	conParam = []string{"-H", "127.0.0.1:4321", "run", "-itd", "--network", "host",
		"--name", ins.Name, "-v", insStatus.DataPath + ":/data",
		fmt.Sprintf("docker.gf.com.cn/gf-mongodb:%s", ins.Version),
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

	mc.insStatus.LastStatus = "creating"
	mc.insStatus.Status = "created"
	mc.insStatus.PrevOp = "CREATE"
	mc.insStatus.LastUpdate = time.Now()
	mc.mm.ma.Send(&ins)
	return nil

RECOVER:
	os.RemoveAll(dataPath)
	glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", m.Name)
	glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")

	mc.insStatus.LastStatus = "creating"
	mc.insStatus.Status = "error"
	mc.insStatus.LastUpdate = time.Now()
	mc.mm.ma.Send(&ins)
	return DeployErr
}

func startMongo(mc *MongoContext) {
	defer Duration(time.Now(), "startMongo")
}

func stopMongo(mc *MongoContext) {
	defer Duration(time.Now(), "stopMongo")
}

func deleteMongo(mc *MongoContext) {
	defer Duration(time.Now(), "deleteMongo")
}
