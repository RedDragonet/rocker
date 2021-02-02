package main

import (
	"syscall"

	"github.com/RedDragonet/rocker/container"
	_ "github.com/RedDragonet/rocker/nsenter"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

func StopContainer(containerName string) {
	pid, err := container.GetContainerPidByName(containerName)
	if err != nil {
		log.Errorf("GetContainerPidByName %s error %v", containerName, err)
		return
	}

	//KILL 进程
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Errorf("syscall kill %s pid %d error %v", containerName, pid, err)
		return
	}

	err = container.StopContainer(containerName)
	if err != nil {
		log.Errorf("StopContainer %s error %v", containerName, err)
		return
	}
}
