package main

import (
	"time"
)

const (
	MAXTRY = 3
)

var DeployErr error = errors.New("Mongo Deploy Failed")
var OpErr error = errors.New("Invalid Operation on Mongo in current state")

type MongoManager struct {
	ma         *MongoAgent
	contextMap map[string]MongoContext
}

func (mm *MongoManager) Send(ins *MongoInstance) error {

}

func (mm *MongoManager) Recovery() error {
	var oldOp string
	var dirList []string
	for _, ins := range mm.ma.mongoMap {
		switch ins.CurrOp {
		case NOP:
			continue
		case CREATE:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = START
			mm.GO_Hanlde(ins)
			ins.NextOp = oldOp
		case START:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = START
			if !ins.Running {
				mm.GO_Hanlde(ins)
			}
			ins.NextOp = oldOp
		case STOP:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = STOP
			if ins.Running {
				mm.GO_Hanlde(ins)
			}
			ins.NextOp = oldOp
		case DELETE:
			ins.CurrOp = ""
			oldOp = ins.NextOp
			ins.NextOp = DELETE
			mm.GO_Hanlde(ins)
			ins.NextOp = oldOp
		}
		dirList = append(dirList, ins.DataPath)
	}
	Clean(dirList)
}

func (mm *MongoManager) CleanDir(dirList []string) {
}

func (mm *MongoManager) GO_Hanlde(ins *Mongo) error {
	var err error
	switch ins.NextOp {
	case CREATE:
		if ins.Created == "" {
			ins.CurrOp = CREATE
			mm.Send(ins)
			if err = mm.createMongo(ins); err {
				ins.Created = false
				ins.PrevOp = CREATE
				ins.CurrOp = ""
				ins.ValidOp = true
				mm.Send(ins)
				return err
			} else {
				ins.Created = true
				ins.PrevOp = CREATE
				ins.CurrOp = ""
				ins.ValidOp = true
				mm.Send(ins)
				mm.ma.monitorMgr.Register(ins.Name)
				return nil
			}
		} else {
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	case START:
		if !(ins.Running || ins.Deleted || ins.CurrOp) {
			ins.CurrOp = START
			mm.Send(ins)
			for i := 0; i < MAXTRY; i++ {
				if err = mm.startMongo(ins); !err {
					break
				}
			}
			ins.PrevOp = START
			ins.CurrOp = ""
			ins.ValidOp = true
			mm.send(ins)
			return nil
		} else {
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	case STOP:
		if ins.Running && !ins.Deleted && !ins.CurrOp {
			ins.CurrOp = STOP
			mm.Send(ins)
			for i := 0; i < MAXTRY; i++ {
				if err = mm.stopMongo(ins); !err {
					break
				}
			}
			ins.PrevOp = STOP
			ins.CurrOp = ""
			ins.ValidOp = true
			mm.Send(ins)
			return nil
		} else {
			ins.ValidOp = false
			mm.Send(ins)
			return OpErr
		}
	case DELETE:
		if ins.Created && !ins.CurrOp {
			ins.CurrOp = DELETE
			mm.Send(ins)
			if ins.Running {
				mm.stopMongo(ins, true)
			}
			for i := 0; i < MAXTRY; i++ {
				if err = mm.deleteMongo(ins); !err {
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
	var conParam []string
	var cmd *exec.Cmd
	var cmdStdout bytes.Buffer
	var cmdStderr bytes.Buffer
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

	return nil

RECOVER:
	os.RemoveAll(dataPath)
	glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb.conf", m.Name)
	glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")

	return DeployErr
}

func (mm *MongoManager) startMongo(ins *Mongo) error {
}

func (mm *MongoManager) stopMongo(ins *Mongo) error {

}

func (mm *MongoManager) deleteMongo(ins *Mongo) error {

}
