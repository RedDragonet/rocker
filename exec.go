package main

import (
	"fmt"
	"github.com/RedDragonet/rocker/cgroup"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const ENV_EXEC_PID = "CONTAINER_PID"
const ENV_PIPE_COMMAND = "CONTAINER_PIPE_COMMAND"
const ENV_PIPE_PARENT = "CONTAINER_PIPE_PARENT"

func ExecContainer(containerName string, cmdArray []string) {
	parent, parentPipeWrite := container.NewParentProcess(true, true, "", nil, nil, "", "")
	if parent == nil {
		log.Errorf("ExecContainer: 创建父进程失败")
		return
	}

	containerInfo, err := container.GetContainerInfo(containerName)
	if err != nil {
		log.Errorf("ExecContainer: GetContainerInfo %s error %v", containerName, err)
		return
	}

	env := getEnvsByPid(containerInfo.State.Pid)
	env = append(env, ENV_EXEC_PID+"="+strconv.Itoa(containerInfo.State.Pid))
	env = append(env, ENV_PIPE_COMMAND+"=3")

	//新建管道
	childPipeRead, childPipeWrite, err := os.Pipe()

	if err != nil {
		log.Errorf("ExecContainer: 新建管道错误 %v", err)
		return
	}

	//传入由子进程写入的管道
	parent.ExtraFiles = append(parent.ExtraFiles, childPipeWrite)
	env = append(env, ENV_PIPE_PARENT+"=4")

	parent.Env = append(os.Environ(), env...)
	if err := parent.Start(); err != nil {
		log.Infof("ExecContainer: 父进程运行失败")
	}

	log.Info("ExecContainer: 等待子init")
	//等待子init写入
	//等待 Setns 完成
	childPipeWrite.Close()
	childPipeData, err := ioutil.ReadAll(childPipeRead)
	if err != nil {
		log.Errorf("ExecContainer: childPipeData %v", err)
		return
	}
	log.Infof("ExecContainer: 子init结束 %s", childPipeData)

	log.Info("ExecContainer: cgroup初始化")

	////cgroup初始化
	cgroupManager := cgroup.NewCgroupManager(containerInfo.ID)

	res := &subsystem.ResourceConfig{
		MemoryLimit: containerInfo.Config.CGroup.MemoryLimit,
		CpuShare:    containerInfo.Config.CGroup.CpuShare,
		CpuSet:      containerInfo.Config.CGroup.CpuSet,
	}
	err = cgroupManager.Apply(parent.Process.Pid, res)
	if err != nil {
		exitError(err)
	}

	log.Info("ExecContainer: cgroup初始化结束")
	parent.ExtraFiles[0].Close()
	if err := sendInitCommand(cmdArray, parentPipeWrite); err != nil {
		exitError(err)
	}

	log.Infof("ExecContainer: 创建父运行成功，开始等待")
	log.Infof("ExecContainer: 当前进程ID", os.Getpid())

	//EXEC默认 交互模式
	//父进程等待子进程退出
	_ = parent.Wait()

	log.Infof("ExecContainer: 父进程运行结束")
}

func getEnvsByPid(pid int) []string {
	path := fmt.Sprintf("/proc/%d/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Read file %s error %v", path, err)
		return nil
	}
	//env split by \u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
