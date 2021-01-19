package main

import (
	"encoding/json"
	"fmt"
	"github.com/RedDragonet/rocker/container"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"text/tabwriter"
)

func ListContainers() {
	files, err := ioutil.ReadDir(container.DefaultInfoLocation)
	if err != nil {
		log.Errorf("Read dir %s error %v", container.DefaultInfoLocation, err)
		return
	}

	var containers []*container.ContainerInfo
	for _, file := range files {
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("Get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			item.ID[:12],
			item.Name,
			item.State.Pid,
			item.State.String(),
			item.Config.Cmd,
			item.Created.Format("2006-01-02 15:04:05"))
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := path.Join(container.DefaultInfoLocation, containerName)
	configFileDir = path.Join(configFileDir, container.ConfigName)
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("Json unmarshal error %v", err)
		return nil, err
	}

	return &containerInfo, nil
}
