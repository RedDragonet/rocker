package container

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/RedDragonet/rocker/cgroup/subsystem"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

func GetContainerInfo(containerName string) (*ContainerInfo, error) {
	containerName = fixContainerName(containerName)

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

	if err != nil {
		log.Errorf("GetContainerPidByName %s error %v", containerName, err)
		return fmt.Errorf("GetContainerPidByName %s error %v", containerName, err)
	}

	pid := info.State.Pid
	//KILL 进程
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Errorf("syscall kill %s pid %d error %v", containerName, pid, err)
		return fmt.Errorf("syscall kill %s pid %d error %v", containerName, pid, err)
	}

	info.State.Paused = true
	return save(info)
}

func RemoveContainer(containerName string) error {
	info, err := GetContainerInfo(containerName)
	if err != nil {
		return err
	}

	if info.State.Running && !info.State.Paused {
		return fmt.Errorf("请先暂停容器")
	}

	CleanUp(info.ID, info.Config.Volumes)
	return nil
}

//补全容器名称
//a4b8ee7aeb557c8f6d10b = a4b8ee7aeb557c8f6d10b87ada8a7b296774447b66d5b4b271c2d4ff499cba3a
func fixContainerName(containerName string) string {
	dirUrl := path.Join(DefaultInfoLocation, containerName)
	if _, err := os.Stat(dirUrl); os.IsNotExist(err) {
		matched := make([]string, 0)

		filepath.Walk(DefaultInfoLocation, func(nwPath string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}
			nwPathBase := path.Base(nwPath)
			if strings.HasPrefix(nwPathBase, containerName) {
				matched = append(matched, nwPathBase)
			}
			return nil
		})

		if len(matched) == 1 {
			return matched[0]
		}

		if len(matched) > 2 {
			fmt.Println("容器名称混淆，匹配到多个符合到容器")
			os.Exit(0)
		}

		if len(matched) == 0 {
			fmt.Println("容器不存在")
			os.Exit(0)
		}
	}

	return containerName
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

func RecordContainerIP(containerId string, ip net.IP) error {
	info, err := GetContainerInfo(containerId)
	if err != nil {
		log.Errorf("RecordContainerIP 容器不存在 %s", containerId)
		return fmt.Errorf("RecordContainerIP 容器不存在 %s", containerId)
	}

	info.Config.IP = ip
	return save(info)
}

func save(containerInfo *ContainerInfo) error {
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return err
	}
	jsonStr := string(jsonBytes)

	dirUrl := path.Join(DefaultInfoLocation, containerInfo.ID)

	if _, err := os.Stat(dirUrl); os.IsNotExist(err) {
		if err := os.MkdirAll(dirUrl, 0644); err != nil {
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

func DeleteContainerInfo(containerId string) error {
	dirURL := path.Join(DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
		return err
	}
	return nil
}

func CleanPortMapping(cinfo *ContainerInfo) error {
	ip := cinfo.Config.IP
	if ip == nil {
		return fmt.Errorf("CleanPortMapping 未配置IP地址")
	}

	log.Infof("CleanPortMapping IP地址 %s", ip.String())
	if err := cleanPortMapping("PREROUTING", ip.String()); err != nil {
		return err
	}

	if err := cleanPortMapping("OUTPUT", ip.String()); err != nil {
		return err
	}

	return nil
}

func cleanPortMapping(table, ip string) error {
	iptablesCmd := fmt.Sprintf("-t nat -nvL %s --line-number", table)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//err := cmd.Run()
	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer output.Close()

	if err := cmd.Start(); err != nil {
		return err
	}

	rd := bufio.NewReaderSize(output, 256)

	/*
		num   pkts bytes target     prot opt in     out     source               destination
		1        2   120 DNAT       tcp  --  *      lo      0.0.0.0/0            0.0.0.0/0            tcp dpt:80 to:192.168.10.2:80
	*/
	line, err := rd.ReadString('\n')
	for err == nil && line != "" {
		//存在对应ip配置
		if strings.Contains(line, ip) {
			number := strings.TrimSpace(line[:4])
			//删除对应配置
			iptablesCmd := fmt.Sprintf("-t nat -D %s %s", table, number)
			cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}
		line, err = rd.ReadString('\n')
	}
	return cmd.Wait()
}

func CleanUp(containerId string, volumes []string) {
	info, err := GetContainerInfo(containerId)
	if err != nil {
		log.Errorf("CleanUp 容器不存在 %s", containerId)
		return
	}

	if volumes == nil {
		volumes = info.Config.Volumes
	}

	DeleteContainerInfo(containerId)
	UnMountVolumeSlice(containerId, volumes)
	DelDefaultDevice(containerId)
	DelWorkSpace(containerId)
	CleanPortMapping(info)
}
