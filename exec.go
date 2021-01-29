package main

import (
	"github.com/RedDragonet/rocker/container"
	_ "github.com/RedDragonet/rocker/nsenter"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const ENV_EXEC_PID = "CONTAINER_PID"
const ENV_EXEC_CMD = "CONTAINER_CMD"

func ExecContainer(containerName string, comArray []string) {
	pid, err := container.GetContainerPidByName(containerName)
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