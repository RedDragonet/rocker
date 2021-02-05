package container

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/RedDragonet/rocker/cgroup/subsystem"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

func GetContainerPidByName(containerName string) (int, error) {
	dirURL := path.Join(DefaultInfoLocation, containerName)
	configFilePath := path.Join(dirURL, ConfigName)
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("容器 %s 不存在", containerName)
		}
		return 0, err
	}
	var containerInfo ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return 0, err
	}
	return containerInfo.State.Pid, nil
}

func GetContainerInfo(containerName string) (*ContainerInfo, error) {
	configFileDir := path.Join(DefaultInfoLocation, containerName)
	configFileDir = path.Join(configFileDir, ConfigName)
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}
	var containerInfo ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("Json unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}

func StopContainer(containerName string) error {
	info, err := GetContainerInfo(containerName)
	if err != nil {
		return err
	}

	info.State.Paused = true
	return save(info)
}

func RemoveContainer(containerName string) error {
	return DeleteContainerInfo(containerName)
}

func RecordContainerInfo(containerPID int, commandArray []string, containerName, id string, volumeSlice, portMapping []string, res *subsystem.ResourceConfig) (string, error) {
	containerInfo := &ContainerInfo{
		ID: id,
		State: State{
			Pid:     containerPID,
			Running: true,
		},
		Config: Config{
			Cmd:     commandArray,
			Image:   "",
			Volumes: volumeSlice,
			CGroup: CGroupResourceConfig{
				MemoryLimit: res.MemoryLimit,
				CpuShare:    res.CpuShare,
				CpuSet:      res.CpuSet,
			},
			PortMapping: portMapping,
		},
		Created: time.Now(),
		Name:    containerName,
	}

	err := save(containerInfo)
	if err != nil {
		return "", err
	}

	return containerName, nil
}

func save(containerInfo *ContainerInfo) error {
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return err
	}
	jsonStr := string(jsonBytes)

	dirUrl := path.Join(DefaultInfoLocation, containerInfo.Name)

	if _, err := os.Stat(dirUrl); os.IsNotExist(err) {
		if err := os.MkdirAll(dirUrl, 0622); err != nil {
			log.Errorf("Mkdir error %s error %v", dirUrl, err)
			return err
		}
	}
	fileName := path.Join(dirUrl, ConfigName)
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return err
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return err
	}
	return nil
}

func DeleteContainerInfo(containerName string) error {
	dirURL := path.Join(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
		return err
	}
	return nil
}
