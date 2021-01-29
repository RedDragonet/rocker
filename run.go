package main

import (
	"encoding/json"
	"github.com/RedDragonet/rocker/cgroup"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	"github.com/RedDragonet/rocker/pkg/stringid"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
	"time"
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

	recordContainerInfo(parent.Process.Pid, cmdArray, containerName, containerID, volume)

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
		deleteContainerInfo(containerID)
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

func recordContainerInfo(containerPID int, commandArray []string, containerName, id, volume string) (string, error) {
	containerInfo := &container.ContainerInfo{
		ID: id,
		State: container.State{
			Pid:     containerPID,
			Running: true,
		},
		Config: container.Config{
			Cmd:     commandArray,
			Image:   "",
			Volumes: volume,
		},
		Created: time.Now(),
		Name:    containerName,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	dirUrl := path.Join(container.DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		log.Errorf("Mkdir error %s error %v", dirUrl, err)
		return "", err
	}
	fileName := path.Join(dirUrl, container.ConfigName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return "", err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return "", err
	}

	return containerName, nil
}

func deleteContainerInfo(containerId string) {
	dirURL := path.Join(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

//func GetContainerPidByName(containerName string) (int, error) {
//	dirURL := path.Join(container.DefaultInfoLocation, containerName)
//	configFilePath := path.Join(dirURL, container.ConfigName)
//	contentBytes, err := ioutil.ReadFile(configFilePath)
//	if err != nil {
//		return 0, err
//	}
//	var containerInfo container.ContainerInfo
//	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
//		return 0, err
//	}
//	return containerInfo.State.Pid, nil
//}
