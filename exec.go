package main

import (
	"encoding/json"
	"github.com/RedDragonet/rocker/container"
	_ "github.com/RedDragonet/rocker/nsenter"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

const ENV_EXEC_PID = "CONTAINER_PID"
const ENV_EXEC_CMD = "CONTAINER_CMD"

func ExecContainer(containerName string, comArray []string) {
	pid, err := GetContainerPidByName(containerName)
	if err != nil {
		log.Errorf("GetContainerPidByName %s error %v", containerName, err)
		return
	}

	cmdStr := strings.Join(comArray, " ")
	log.Infof("container pid %d", pid)
	log.Infof("container command %s", cmdStr)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if pid > 0 {
		os.Setenv(ENV_EXEC_PID, strconv.Itoa(pid))
	} else {
		os.Setenv(ENV_EXEC_PID, "")
	}

	os.Setenv(ENV_EXEC_CMD, cmdStr)

	if err := cmd.Run(); err != nil {
		log.Errorf("Exec container %s error %v", containerName, err)
	}
}

func GetContainerPidByName(containerName string) (int, error) {
	dirURL := path.Join(container.DefaultInfoLocation, containerName)
	configFilePath := path.Join(dirURL, container.ConfigName)
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return 0, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return 0, err
	}
	return containerInfo.State.Pid, nil
}