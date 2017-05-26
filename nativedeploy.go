package main

import (
	"errors"
	"fmt"
	cfgTmpl "github.com/GBjuno/dbagent/template"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"teego/pkg/api"
	"text/template"
	"time"
	//"path"
	"strconv"
)

const (
	binPath = "/usr/local/mongo-%s"
)

type NativeDeployment struct {
	mm *MongoManager
}

func NewNativeDeploymenet(mm *MongoManager) *NativeDeployment {
	return &NativeDeployment{mm: mm}
}

//func (n *NativeDeployment) stop(ins *Mongo) do nothing with create
func (n *NativeDeployment) createMongo(ins *api.MongoInstance) error {
	defer Duration(time.Now(), "NATIVE_createMongo")

	var err error
	var osVer string
	var initScript string
	var initPath string
	var f *os.File
	var initF *os.File
	var tmpl *template.Template
	var tmplConf string
	var now time.Time = time.Now()
	var dataPath string = fmt.Sprintf("%s/%s_%04d%02d%02d_%02d%02d%04d", ins.Status.BasePath, ins.GetName(),
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Nanosecond())

	if _, err := os.Stat(ins.Status.BasePath); os.IsNotExist(err) {
		glog.Errorf("BasePath %s does not exist", ins.Status.BasePath)
		return DeployErr
	}

	//create mongo datapath
	if err = os.Mkdir(dataPath, os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath)
		return DeployErr
	}
	user, _ := user.Lookup("mongodb")
	uid, _ := strconv.Atoi(user.Uid)
	gid, _ := strconv.Atoi(user.Gid)

	if err = os.Chown(dataPath, uid, gid); err != nil {
		glog.Errorf("can not chown %s, uid %d, gid %d", dataPath, uid, gid)
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath, ins.GetName())

	if err = os.Mkdir(dataPath+"/mongodb-"+ins.GetName(), os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath)
		return DeployErr
	}

	if err = os.Chown(dataPath+"/mongodb-"+ins.GetName(), uid, gid); err != nil {
		glog.Errorf("can not chown %s", dataPath)
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath+"/mongodb-"+ins.GetName(), ins.GetName())

	ins.Status.DataPath = dataPath

	//create configuration file
	glog.Infof("creating configuration file mongodb.conf for mongo %s", ins.GetName())
	f, err = os.OpenFile(dataPath+"/mongodb-"+ins.GetName()+".conf", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	if err != nil {
		glog.Errorf("can not create configuration file %s", dataPath+"/mongodb-"+ins.GetName()+".conf")
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb.conf")
		return DeployErr
	}

	if err = os.Chown(dataPath+"/mongodb-"+ins.GetName()+".conf", uid, gid); err != nil {
		glog.Errorf("can not chown %s", dataPath)
		return DeployErr
	}

	if ins.Spec.Replication == "" {
		tmplConf = cfgTmpl.NATIVE_Single
	} else {
		tmplConf = cfgTmpl.NATIVE_Replset
	}

	tmpl, err = template.New("db").Parse(tmplConf)
	if err != nil {
		glog.Errorf("can not template %d, err %v", tmplConf, err)
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}

	err = tmpl.Execute(f, ins)
	if err != nil {
		glog.Errorf("can not template %d, err %v", tmplConf, err)
		os.RemoveAll(dataPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}
	glog.Infof("create file %s for mongo %s", dataPath+"/mongodb.conf", ins.GetName())

	pwd, _ := os.Getwd()

	if _, err = os.Stat("/etc/redhat-release"); os.IsNotExist(err) {
		osVer = "ubuntu"
		initScript = filepath.Join(pwd, "./template/startupscript_ubuntu")
		initPath = "/etc/init/mongodb-" + ins.GetName() + ".conf"
	} else {
		osVer = "centos"
		initPath = "/etc/init/mongodb-" + ins.GetName()
		initScript = filepath.Join(pwd, "./template/startupscript_centos")
		initPath = "/etc/init.d/mongodb-" + ins.GetName()
		if _, err = os.Stat("/usr/bin/systemctl"); os.IsNotExist(err) {
			glog.Infof("os does not use systemd")
		} else {
			glog.Infof("os uses systemd, reloading initscript conf")
			cmd := exec.Command("systemctl", "daemon-reload")
			cmd.Start()
		}
	}

	if err = n.getMongoBinary(ins.Spec.Version, osVer); err != nil {
		glog.Errorf("can not get binary file, version %s, err %v", ins.Spec.Version, err)
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
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}

	err = tmpl.Execute(initF, ins)
	if err != nil {
		glog.Errorf("can not template startup script %s, err %v", initScript, err)
		os.RemoveAll(dataPath)
		os.RemoveAll(initPath)
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "/mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}

	glog.Infof("create startup script %s for mongo %s", initPath, ins.GetName())

	return nil
}

//func (n *nativceDeploy) start(*Mongo) (interface{}, error) is used to deploy mongodb using binary
func (n *NativeDeployment) startMongo(ins *api.MongoInstance) error {
	mconf := getMongoConfFromMongoInstance(ins)
	cmd := exec.Command("service", "mongodb-"+ins.GetName(), "start")
	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Infof("start mongo instance %s failed, output %s", ins.GetName(), out)
		return err
	}
	glog.Infof("start mongo instance %s succeed, output %s", ins.GetName(), out)

	/*
		cmd := exec.Command(fmt.Sprintf(binPath,mconf.Version)+"/mongod", "-f", mconf.DataPath+"/mongodb-"+ins.GetName()+".conf")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}
		glog.Infof("start mongo instance %s succeed, output %s", ins.GetName(), out)
	*/
	time.Sleep(time.Duration(3) * time.Second)
	file, err := os.Open(mconf.DataPath + "/mongodb-" + ins.GetName() + ".pid")
	if err != nil {
		glog.Errorf("cannot get the pidfile of mongo instance %s, path %s, err %s", ins.GetName(), mconf.DataPath+"/mongodb-"+ins.GetName()+".pid", err)
		return err
	}

	pid := make([]byte, 100)
	count, err := file.Read(pid)
	if err != nil {
		return err
	}

	ins.Status.Pid = string(pid[0 : count-1])
	return nil
}

//func (n *NativeDeployment) stop(ins *Mongo) is used to kill mongodb process
func (n *NativeDeployment) stopMongo(ins *api.MongoInstance) error {
	if err := shutdownMongo(ins, false); err != nil {
		glog.Errorf("stop mongo instance %s failed, output", ins.GetName())
		return err
	}
	glog.Errorf("stop mongo instance %s succeed, output", ins.GetName())
	return nil
	/*
		mconf := getMongoConfFromMongoInstance(ins)
		cmd := exec.Command("service", "mongodb-"+ins.GetName(), "stop")
		out, err := cmd.CombinedOutput()
		if err != nil {
			glog.Errorf("stop mongo instance %s failed, output %s", ins.GetName(), out)
			return err
		}
		glog.Infof("stop mongo instance %s succeed, output %s", ins.GetName(), out)
		return nil
	*/
}

//func (n *NativeDeployment) stop(ins *Mongo) do nothing with delete
func (n *NativeDeployment) deleteMongo(ins *api.MongoInstance) error {
	if err := n.stopMongo(ins); err != nil {
		if err = n.killMongo(ins); err != nil {
			glog.Infof("delete mongo instance %s failed", ins.GetName())
			return err
		}
	}
	initScript := fmt.Sprintln("/etc/init/mongodb-%s.conf", ins.GetName())
	os.RemoveAll(ins.Status.DataPath)
	os.RemoveAll(initScript)
	glog.Infof("delete mongo instance %s succeed", ins.GetName())
	glog.Infof("remove mongo instance directory %s", ins.Status.DataPath)
	return nil
}

func (n *NativeDeployment) killMongo(ins *api.MongoInstance) error {

	cmd := exec.Command("kill", "-9", ins.Status.Pid)
	out, _ := cmd.CombinedOutput()
	glog.Infof("kill -9 mongo instance %s succeed, output %s", ins.GetName(), out)
	return nil
}

func (n *NativeDeployment) getMongoBinary(mongoVer string, osVer string) error {
	binUrl := fmt.Sprintf("http://10.2.86.104/mongo-source/mongodb-%s-%s.tar.gz", osVer, mongoVer)
	f := fmt.Sprintf("/tmp/mongodb-%s-%s.tar.gz", osVer, mongoVer)
	binPath := fmt.Sprintf(binPath, mongoVer)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		resp, err := http.Get(binUrl)
		if err != nil {
			glog.Errorf("can not download mongo binary %s, err %v", binPath, err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			glog.Errorf("can not download mongo binary %s, status %s, err %v", binPath, resp.StatusCode, err)
			return errors.New(fmt.Sprintf("download mongo binary failed, url %s, status %s", binUrl, resp.StatusCode))
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err = ioutil.WriteFile(f, body, 0644); err != nil {
			glog.Errorf("can not save download mongo binary %s, err %v", f, err)
			return err
		}
		glog.Errorf("save download mongo binary %s", f)

		os.Chdir("/usr/local/")
		cmd := exec.Command("tar", "-xf", f)
		output, err := cmd.CombinedOutput()
		if err != nil {
			os.RemoveAll(f)
			glog.Errorf("cannot gunzip file %s, output %s, err %s", f, output, err)
			return err
		}
		glog.Infof("gunzip file %s, output %s", f, output)

		cmd = exec.Command("mv", fmt.Sprintf("/usr/local/mongodb-%s-%s", osVer, mongoVer), binPath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			os.RemoveAll(f)
			os.RemoveAll(fmt.Sprintf("/usr/local/mongodb-%s-%s", osVer, mongoVer))
			glog.Errorf("cannot move to /usr/local, output %s, err %s", output, err)
			return err
		}
		glog.Infof("mongo %s binary is setup", mongoVer)
	} else {
		glog.Infof("%s is already setup", binPath)
		return nil
	}
	return nil
}
