package main

import (
	"fmt"
	cfgTmpl "github.com/GBjuno/dbagent/template"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerCli "github.com/docker/docker/client"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"os"
	"teego/pkg/api"
	"text/template"
	"time"
)

type DockerDeployment struct {
	mm      *MongoManager
	client  *dockerCli.Client
	timeout time.Duration
}

func NewDockerDeployment(mm *MongoManager) *DockerDeployment {
	defer Duration(time.Now(), "NewDockerDeployment")
	var dockerEndpoint string
	client, err := getDockerClient(dockerEndpoint)
	if err != nil {
		glog.Errorf("Couldn't connect to docker: %v", err)
	}
	glog.Infof("Start docker client with request timeout 10s")
	dockerDeploy := &DockerDeployment{mm: mm, client: client, timeout: time.Duration(10) * time.Second}
	return dockerDeploy
}

func getDockerClient(dockerEndpoint string) (*dockerCli.Client, error) {
	if len(dockerEndpoint) > 0 {
		glog.Infof("Connecting to docker on %s", dockerEndpoint)
		return dockerCli.NewClient(dockerEndpoint, "", nil, nil)
	}
	return dockerCli.NewEnvClient()
}

func newMongoContainerConfig(mconf *MongoConf) (*container.Config, *container.HostConfig) {
	return &container.Config{
			Image: fmt.Sprintf("docker.gf.com.cn/gf-mongodb:%s", mconf.Version),
			Cmd:   []string{"bash", "-c", "/usr/bin/mongod -f /data/mongodb-" + mconf.Name + ".conf"},
		}, &container.HostConfig{
			NetworkMode: "host",
			Binds:       []string{fmt.Sprintf("%s:/data:rw", mconf.DataPath), "/etc/localtime:/etc/localtime:ro"},
		}
}

func (dockerDeploy *DockerDeployment) getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), dockerDeploy.timeout)
}

func (dockerDeploy *DockerDeployment) createMongo(ins *api.MongoInstance) error {
	defer Duration(time.Now(), "DOCKER_createMongo")

	var err error
	var f *os.File
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

	if err = os.Mkdir(dataPath+"/mongodb-"+ins.GetName(), os.ModeDir|0755); err != nil {
		glog.Errorf("can not mkdir %s", dataPath+"/mongodb-"+ins.GetName())
		return DeployErr
	}

	glog.Infof("create directory %s for mongo %s", dataPath, ins.GetName())
	ins.Status.DataPath = dataPath

	//create configuration file
	glog.Infof("creating configuration file mongodb.conf for mongo %s", ins.GetName())
	f, err = os.OpenFile(dataPath+"/mongodb-"+ins.GetName()+".conf", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	if err != nil {
		glog.Errorf("can not create configuration file %s", dataPath+"/mongodb-"+ins.GetName()+".conf")
		os.RemoveAll(dataPath)
		glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb-"+ins.GetName()+".conf", ins.GetName())
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}

	if ins.Spec.Replication == "" {
		tmplConf = cfgTmpl.DOCKER_Single
	} else {
		tmplConf = cfgTmpl.DOCKER_Replset
	}

	tmpl, err = template.New("db").Parse(tmplConf)
	if err != nil {
		glog.Errorf("can not template %d", tmplConf)
		os.RemoveAll(dataPath)
		glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb-"+ins.GetName()+".conf", ins.GetName())
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}

	err = tmpl.Execute(f, ins)
	if err != nil {
		glog.Errorf("can not template %d", tmplConf)
		os.RemoveAll(dataPath)
		glog.Infof("template file %s for mongo %s failed", dataPath+"/mongodb-"+ins.GetName()+".conf", ins.GetName())
		glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb-"+ins.GetName()+".conf")
		return DeployErr
	}
	glog.Infof("create file %s for mongo %s", dataPath+"/mongodb.conf", ins.GetName())

	//startup the mongodb mongo instance by using docker
	resp, err := dockerDeploy.createContainer(ins)
	if err != nil {
		glog.Errorf("run docker container failed: %s", err.Error())
		goto RECOVER
	}

	ins.Status.Pid = resp.ID
	dockerDeploy.mm.ma.GO_UpdateMongoInstance(ins)
	glog.Infof("create docker container succeed, id: %s", ins.Status.Pid)
	return nil

RECOVER:
	os.RemoveAll(dataPath)
	glog.Infof("remove dir %s and file %s to recover status", dataPath, "mongodb-"+ins.GetName()+".conf")

	return DeployErr
}

func (dockerDeploy *DockerDeployment) startMongo(ins *api.MongoInstance) error {
	defer Duration(time.Now(), "DOCKER_startMongo")

	glog.Infof("starting mongo instance %s, container id %s", ins.GetName(), ins.Status.Pid)
	if err := dockerDeploy.startContainer(ins.Status.Pid); err != nil {
		glog.Errorf("start mongo instance %s and container id %s failed, err: %v", ins.GetName(), ins.Status.Pid, err)
		return err
	}

	glog.Infof("start mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)
	go dockerDeploy.mm.ma.monitorMgr.simpleCheckOneIns(ins.GetName())
	return nil
}

func (dockerDeploy *DockerDeployment) stopMongo(ins *api.MongoInstance) error {
	defer Duration(time.Now(), "DOCKER_stopMongo")
	force := false

	glog.Infof("stopping mongo instance %s, container id %s", ins.GetName(), ins.Status.Pid)
	if err := shutdownMongo(ins, force); err != nil {
		glog.Infof("stop mongo instance %s, container id %s failed, err: %v", ins.GetName(), ins.Status.Pid, err)
		return err
	}
	/*
		glog.Infof("stopping mongo %s, container id %s", ins.GetName(), ins.Status.Pid)
		if err := dockerMgr.stopContainer(ins.Status.Pid); err != nil {
			glog.Errorf("stop mongo %s and container id %s failed", ins.GetName(), ins.Status.Pid)
			return err
		}
	*/
	glog.Infof("stop mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)
	go dockerDeploy.mm.ma.monitorMgr.simpleCheckOneIns(ins.GetName())
	return nil
}

func (dockerDeploy *DockerDeployment) deleteMongo(ins *api.MongoInstance) error {
	defer Duration(time.Now(), "DOCKER_deleteMongo")

	glog.Infof("stopping mongo instance %s, container id %s", ins.GetName(), ins.Status.Pid)
	if err := dockerDeploy.stopContainer(ins.Status.Pid); err != nil {
		glog.Errorf("stop mongo instance %s and container id %s failed, err: %v", ins.GetName(), ins.Status.Pid, err)
		return err
	}
	glog.Infof("stop mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)

	glog.Infof("removing mongo instance %s, container id %s", ins.GetName(), ins.Status.Pid)
	if err := dockerDeploy.removeContainer(ins.Status.Pid); err != nil {
		glog.Errorf("remove mongo %s and container id %s failed, err: %v", ins.GetName(), ins.Status.Pid, err)
		return err
	}
	glog.Infof("remove mongo instance %s, container id %s succeed", ins.GetName(), ins.Status.Pid)

	os.RemoveAll(ins.Status.DataPath)
	glog.Infof("remove mongo instance directory %s", ins.Status.DataPath)
	return nil

}

func (dockerDeploy *DockerDeployment) createContainer(ins *api.MongoInstance) (*container.ContainerCreateCreatedBody, error) {
	mconf := getMongoConfFromMongoInstance(ins)
	ctx, cancel := dockerDeploy.getTimeoutContext()
	defer cancel()
	config, hostConfig := newMongoContainerConfig(mconf)
	createResp, err := dockerDeploy.client.ContainerCreate(ctx, config, hostConfig, nil, mconf.Name)
	if err != nil {
		return nil, err
	}
	return &createResp, nil
}

func (dockerDeploy *DockerDeployment) startContainer(containerID string) error {
	ctx, cancel := dockerDeploy.getTimeoutContext()
	defer cancel()
	if err := dockerDeploy.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	return nil
}

func (dockerDeploy *DockerDeployment) stopContainer(containerID string) error {
	ctx, cancel := dockerDeploy.getTimeoutContext()
	defer cancel()
	if err := dockerDeploy.client.ContainerStop(ctx, containerID, &dockerDeploy.timeout); err != nil {
		return err
	}
	return nil
}

func (dockerDeploy *DockerDeployment) killContainer(containerID string) error {
	ctx, cancel := dockerDeploy.getTimeoutContext()
	defer cancel()
	if err := dockerDeploy.client.ContainerKill(ctx, containerID, "SIGKILL"); err != nil {
		return err
	}
	return nil
}

func (dockerDeploy *DockerDeployment) removeContainer(containerID string) error {
	ctx, cancel := dockerDeploy.getTimeoutContext()
	defer cancel()
	if err := dockerDeploy.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{RemoveVolumes: false, RemoveLinks: false, Force: true}); err != nil {
		return err
	}
	return nil
}
