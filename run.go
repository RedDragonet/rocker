package main

import (
	"github.com/RedDragonet/rocker/cgroup"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	stringid2 "github.com/RedDragonet/rocker/pkg/stringid"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strings"
	"time"
)

func Run(interactive, tty bool, volume string, cmdArray []string, res *subsystem.ResourceConfig) {
	containerId := stringid2.GenerateRandomID()

	parent, pipeWrite := container.NewParentProcess(interactive, tty, cmdArray[0], volume, containerId)
	if parent == nil {
		log.Errorf("创建父进程失败")
		return
	}

	containerID := randStringBytes(10)

	log.Infof("当前进程ID", os.Getpid())

	if err := parent.Start(); err != nil {
		log.Infof("父进程运行失败")
	}

	if err := sendInitCommand(cmdArray[1:], pipeWrite); err != nil {
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

	//交互模式
	//父进程等待子进程退出
	if interactive {
		_ = parent.Wait()
	}

	container.UnMountVolume(containerId, volume)
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
