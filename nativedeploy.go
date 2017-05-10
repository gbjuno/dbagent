package main

import (
	"fmt"
	cfgTmpl "github.com/GBjuno/dbagent/template"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"text/template"
	"time"
	//"path"
	//"strconv"
)

const (
	binPath = "/usr/local/mongo-%s/"
)

type NativeDeployment struct {
	mm *MongoManager
}

func NewNativeDeploymenet(mm *MongoManager) *NativeDeployment {
	return &NativeDeployment{mm: mm}
}

//func (n *NativeDeployment) stop(ins *Mongo) do nothing with create
func (n *NativeDeployment) createMongo(ins *Mongo) error {
	mconf := getMongoConfFromMongoInstance(ins)
	defer Duration(time.Now(), "NATIVE_createMongo")

	var err error
	var osVer string
	var initScript string
	var initPath string
	var f *os.File
	var initF *os.File
	var binF *os.File
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

	if err = os.Mkdir(dataPath+"/mongodb-"+ins.Name, os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath)
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath, ins.Name)

	ins.DataPath = dataPath

	//create configuration file
	glog.Infof("creating configuration file mongodb.conf for mongo %s", ins.Name)
	f, err = os.OpenFile(dataPath+"/mongodb-"+ins.Name+".conf", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	if err != nil {
		glog.Errorf("can not create configuration file %s", dataPath+"/mongodb-"+ins.Name+".conf")
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}

	switch ins.Type {
	case SingleDB:
		tmplConf = cfgTmpl.NATIVE_Single
	case ReplsetDB:
		tmplConf = cfgTmpl.NATIVE_Replset
	default:
		tmplConf = cfgTmpl.NATIVE_Single
	}

	tmpl, err = template.New("db").Parse(tmplConf)
	if err != nil {
		glog.Errorf("can not template %d, err %v", tmplConf, err)
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.Name+".conf")
		return DeployErr
	}

	err = tmpl.Execute(f, ins)
	if err != nil {
		glog.Errorf("can not template %d, err %v", tmplConf, err)
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.Name+".conf")
		return DeployErr
	}
	glog.Infof("create file %s for mongo %s", dataPath+"/mongodb.conf", ins.Name)

	if _, err = os.Stat("/etc/redhat-release"); os.IsNotExist(err) {
		osVer = "ubuntu"
		initScript = "template/startupscript_ubuntu"
		initPath = "/etc/init/mongodb-" + ins.Name
	} else {
		osVer = "centos"
		initScript = "template/startupscript_centos"
		initPath = "/etc/init.d/mongodb-" + ins.Name
	}

	if err = n.getMongoBinary(ins.Version, osVer); err != nil {
		glog.Errorf("can not get binary file, version %s, err %v", ins.Version, err)
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}

	initF, err = os.OpenFile(initPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		glog.Errorf("can not create startup script %s, err %v", initPath, err)
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}
	defer initF.Close()

	tmpl, err = template.ParseFiles(initScript)
	if err != nil {
		glog.Errorf("can not template startup script %s, err %v", initScript, err)
		os.RemoveAll(dataPath)
		os.RemoveAll(initPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.Name+".conf")
		return DeployErr
	}

	err = tmpl.Execute(initF, ins)
	if err != nil {
		glog.Errorf("can not template startup script %s, err %v", initScript, err)
		os.RemoveAll(dataPath)
		os.RemoveAll(initPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.Name+".conf")
		return DeployErr
	}

	glog.Infof("create startup script %s for mongo %s", initPath, ins.Name)

	return nil
}

//func (n *nativceDeploy) start(*Mongo) (interface{}, error) is used to deploy mongodb using binary
func (n *NativeDeployment) startMongo(ins *Mongo) error {
	mconf := getMongoConfFromMongoInstance(ins)
	cmd := exec.Command("service", "mongodb-"+ins.Name, "start")
	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Infof("start mongo instance %s failed, output %s", ins.Name, out)
		return err
	}
	glog.Infof("start mongo instance %s succeed, output %s", ins.Name, out)

	/*
		cmd := exec.Command(fmt.Sprintf(binPath,mconf.Version)+"mongod", "-f", mconf.DataPath+"/mongodb-"+ins.Name+".conf")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}
		glog.Infof("start mongo instance %s succeed, output %s", ins.Name, out)
	*/

	file, err := os.Open(mconf.DataPath + "/mongodb-" + ins.Name + ".pid")
	if err != nil {
		glog.Errorf("cannot get the pidfile of mongo instance %s, err %s", ins.Name, err)
		return err
	}

	pid := make([]byte, 100)
	_, err = file.Read(pid)
	if err != nil {
		return err
	}

	ins.ContainerID = string(pid)
	return nil
}

//func (n *NativeDeployment) stop(ins *Mongo) is used to kill mongodb process
func (n *NativeDeployment) stopMongo(ins *Mongo) error {
	if err := shutdownMongo(ins, false); err != nil {
		glog.Errorf("stop mongo instance %s failed, output", ins.Name)
		return err
	}
	glog.Errorf("stop mongo instance %s succeed, output", ins.Name)
	return nil
	/*
		mconf := getMongoConfFromMongoInstance(ins)
		cmd := exec.Command("service", "mongodb-"+ins.Name, "stop")
		out, err := cmd.CombinedOutput()
		if err != nil {
			glog.Errorf("stop mongo instance %s failed, output %s", ins.Name, out)
			return err
		}
		glog.Infof("stop mongo instance %s succeed, output %s", ins.Name, out)
		return nil
	*/
}

//func (n *NativeDeployment) stop(ins *Mongo) do nothing with delete
func (n *NativeDeployment) deleteMongo(ins *Mongo) error {
	if err := n.killMongo(ins); err != nil {
		glog.Infof("delete mongo instance %s failed", ins.Name)
		return err
	}
	os.RemoveAll(ins.DataPath)
	glog.Infof("delete mongo instance %s succeed", ins.Name)
	glog.Infof("remove mongo instance directory %s", ins.DataPath)
	return nil
}

func (n *NativeDeployment) killMongo(ins *Mongo) error {
	mconf := getMongoConfFromMongoInstance(ins)
	file, err := os.Open(mconf.DataPath + "/mongodb-" + ins.Name + ".pid")
	if err != nil {
		return err
	}

	pid := make([]byte, 100)
	_, err = file.Read(pid)
	if err != nil {
		return err
	}

	cmd := exec.Command("kill", "-9", string(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("kill -9 mongo instance %s failed, output %s, err %v", ins.Name, out, err)
		return err
	}

	glog.Infof("kill -9 mongo instance %s succeed, output %s", ins.Name, out)
	return nil
}

func (n *NativeDeployment) getMongoBinary(mongoVer string, osVer string) error {
	binUrl = fmt.Sprintf("10.2.86.104/mongo-source/mongodb-%s-%s.tar.gz", osVer, mongoVer)
	if err := os.Stat(fmt.Sprintf(binPath, mongoVer)); os.IsExist(err) {
		retur
		resp, err := http.Get(binUrl)
		if err != nil {
			glog.Errorf("can not download mongo binary %s, err %v", binPath, err)
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		f := fmt.Sprintf("/tmp/mongodb-%s-%s.tar.gz", osVer, mongoVer)
		if err = ioutil.WriteFile(f, body, 0755); err != nil {
			glog.Errorf("can not save download mongo binary %s, err %v", f, err)
			return err
		}
	}

	return nil
}
