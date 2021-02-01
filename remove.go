package main

import (
	"github.com/RedDragonet/rocker/container"
	_ "github.com/RedDragonet/rocker/nsenter"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

func RemoveContainer(containerName string) {
	err := container.RemoveContainer(containerName)
	if err != nil {
		log.Errorf("RemoveContainer %s error %v", containerName, err)
		return
	}
}
