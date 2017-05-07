package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerCli "github.com/docker/docker/client"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"time"
)

type MongoConf struct {
	Name     string
	Env      []string
	DataPath string
	Version  string
}

func getMongoConfFromMongoInstance(m *Mongo) *MongoConf {
	conf := MongoConf{Name: m.Name, DataPath: m.DataPath, Version: m.Version}
	return &conf
}

type DockerManager struct {
	client  *dockerCli.Client
	timeout time.Duration
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
			Cmd:   []string{"bash", "-c", "/usr/bin/mongod -f /data/mongodb.conf"},
		}, &container.HostConfig{
			NetworkMode: "host",
			Binds:       []string{fmt.Sprintf("%s:/data:rw", mconf.DataPath), "/etc/localtime:/etc/localtime:ro"},
		}
}

var dockerMgr *DockerManager

func NewDockerManager() *DockerManager {
	var dockerEndpoint string
	Duration(time.Now(), "NewDockerManager")
	once.Do(func() {
		client, err := getDockerClient(dockerEndpoint)
		if err != nil {
			glog.Fatalf("Couldn't connect to docker: %v", err)
		}
		glog.Infof("Start docker client with request timeout 10s")
		dockerMgr = &DockerManager{client: client, timeout: time.Duration(10) * time.Second}
	})
	return dockerMgr
}

func (dockerMgr *DockerManager) getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), dockerMgr.timeout)
}

func (dockerMgr *DockerManager) createContainer(m *Mongo) (*container.ContainerCreateCreatedBody, error) {
	mconf := getMongoConfFromMongoInstance(m)
	ctx, cancel := dockerMgr.getTimeoutContext()
	defer cancel()
	config, hostConfig := newMongoContainerConfig(mconf)
	createResp, err := dockerMgr.client.ContainerCreate(ctx, config, hostConfig, nil, mconf.Name)
	if err != nil {
		return nil, err
	}
	return &createResp, nil
}

func (dockerMgr *DockerManager) startContainer(containerID string) error {
	ctx, cancel := dockerMgr.getTimeoutContext()
	defer cancel()
	if err := dockerMgr.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	return nil
}

func (dockerMgr *DockerManager) stopContainer(containerID string) error {
	ctx, cancel := dockerMgr.getTimeoutContext()
	defer cancel()
	if err := dockerMgr.client.ContainerStop(ctx, containerID, &dockerMgr.timeout); err != nil {
		return err
	}
	return nil
}

func (dockerMgr *DockerManager) killContainer(containerID string) error {
	ctx, cancel := dockerMgr.getTimeoutContext()
	defer cancel()
	if err := dockerMgr.client.ContainerKill(ctx, containerID, "SIGKILL"); err != nil {
		return err
	}
	return nil
}

func (dockerMgr *DockerManager) removeContainer(containerID string) error {
	ctx, cancel := dockerMgr.getTimeoutContext()
	defer cancel()
	if err := dockerMgr.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: true, Force: true}); err != nil {
		return err
	}
	return nil
}
