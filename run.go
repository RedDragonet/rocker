package main

import (
	"github.com/RedDragonet/rocker/cgroup"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	stringid "github.com/RedDragonet/rocker/pkg"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strings"
	"time"
)

func Run(interactive, tty bool, cmdArray []string, res *subsystem.ResourceConfig) {
	containerId := stringid.GenerateRandomID()

	parent, pipeWrite := container.NewParentProcess(interactive, tty, containerId)
	if parent == nil {
		log.Errorf("创建父进程失败")
		return
	}

	containerID := randStringBytes(10)

	log.Infof("当前进程ID", os.Getpid())

	if err := parent.Start(); err != nil {
		log.Infof("父进程运行失败")
	}

	if err := sendInitCommand(cmdArray, pipeWrite); err != nil {
		exitError(err)
	}

	log.Infof("创建父运行成功，开始等待")
	log.Infof("当前进程ID", os.Getpid())

	//cgroup初始化
	cgroupManager := cgroup.NewCgroupManager(containerID)
	defer cgroupManager.Destroy()
	err := cgroupManager.Set(res)
	if err != nil {
		exitError(err)
	}

	err = cgroupManager.Apply(parent.Process.Pid, res)
	if err != nil {
		exitError(err)
	}

	_ = parent.Wait()

	container.DelWorkSpace(containerId)

	log.Infof("父进程运行结束")

	os.Exit(0)
}

func sendInitCommand(cmdArray []string, pipeWrite *os.File) (err error) {
	args := strings.Join(cmdArray, " ")
	log.Infof("发送初始化参数 %s", args)
	_, err = pipeWrite.WriteString(args)
	pipeWrite.Close()
	return
}

func exitError(err error) {
	log.Errorf("系统异常", err)
	os.Exit(0)
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
