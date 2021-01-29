package main

import (
	"github.com/RedDragonet/rocker/cgroup"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	"github.com/RedDragonet/rocker/pkg/stringid"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func Run(interactive, tty bool, volume string, cmdArray []string, res *subsystem.ResourceConfig, containerName string) {
	containerID := stringid.GenerateRandomID()
	if containerName == "" {
		containerName = containerID[:12]
	}

	parent, pipeWrite := container.NewParentProcess(interactive, tty, cmdArray[0], volume, containerID, containerName)
	if parent == nil {
		log.Errorf("创建父进程失败")
		return
	}

	log.Infof("当前进程ID", os.Getpid())

	if err := parent.Start(); err != nil {
		log.Infof("父进程运行失败")
	}

	container.RecordContainerInfo(parent.Process.Pid, cmdArray, containerName, containerID, volume)

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
		container.DeleteContainerInfo(containerName)
		container.UnMountVolume(containerID, volume)
		container.DelWorkSpace(containerID)
	}

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
