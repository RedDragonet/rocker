package main

import (
	"github.com/RedDragonet/rocker/container"
	_ "github.com/RedDragonet/rocker/nsenter"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

func StopContainer(containerName string) {
	err := container.StopContainer(containerName)
	if err != nil {
		log.Errorf("StopContainer %s error %v", containerName, err)
		return
	}
}
