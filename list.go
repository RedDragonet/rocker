package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	"github.com/RedDragonet/rocker/container"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

func ListContainers() {
	files, err := ioutil.ReadDir(container.DefaultInfoLocation)
	if err != nil {
		log.Errorf("Read dir %s error %v", container.DefaultInfoLocation, err)
		return
	}

	var containers []*container.ContainerInfo
	for _, file := range files {
		tmpContainer, err := container.GetContainerInfo(file.Name())
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
