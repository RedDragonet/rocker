package main

import (
	"github.com/RedDragonet/rocker/image"
	"github.com/RedDragonet/rocker/network"
	"os"
	"os/exec"
	"strings"

	"github.com/RedDragonet/rocker/cgroup"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"github.com/RedDragonet/rocker/pkg/stringid"
)

const DEFAULT_BRIDGE = "rocker0"

func Run(interactive, tty bool, net string, volumes, portMapping, environ, argv []string, res *subsystem.ResourceConfig, containerName string) {
	containerID := stringid.GenerateRandomID()
	var retErr error

	defer func() {
		if retErr != nil {
			log.Errorf("启动失败 %v", retErr)
			container.StopContainer(containerID)
			container.RemoveContainer(containerID)
		}
	}()

	if containerName == "" {
		containerName = containerID[:12]
	}

	image.Init()
	parent, pipeWrite := container.NewParentProcess(interactive, tty, argv[0], volumes, environ, containerID, containerName)
	if parent == nil {
		log.Errorf("创建父进程失败")
		return
	}

	log.Infof("当前进程ID %d ", os.Getpid())

	if err := parent.Start(); err != nil {
		log.Infof("父进程运行失败")
	}

	container.RecordContainerInfo(parent.Process.Pid, argv, containerName, containerID, volumes, portMapping, res)

	//cgroup初始化
	cgroupManager := cgroup.NewCgroupManager(containerID)
	defer cgroupManager.Destroy()
	err := cgroupManager.Set(res)
	if err != nil {
		retErr = err
		return
	}

	err = cgroupManager.Apply(parent.Process.Pid, res)
	if err != nil {
		retErr = err
		return
	}

	//创建默认设备
	if len(portMapping) > 0 && net == "" {
		net = DEFAULT_BRIDGE
		createDefaultBridge()
	}

	if net != "" {
		// 配置网络
		if err := network.Init(); err != nil {
			retErr = err
			return
		}
		containerInfo := &container.ContainerInfo{
			ID: containerID,
			State: container.State{
				Pid: parent.Process.Pid,
			},
			Name: containerName,
			Config: container.Config{
				PortMapping: portMapping,
			},
		}
		if err := network.Connect(net, containerInfo); err != nil {
			log.Errorf("Error Connect Network %v", err)
			retErr = err
			return
		}
	}

	if err := sendInitCommand(argv[1:], pipeWrite); err != nil {
		retErr = err
		return
	}

	log.Infof("创建父运行成功，开始等待")
	log.Infof("当前进程ID %d ", os.Getpid())

	//交互模式
	//父进程等待子进程退出
	if interactive {
		_ = parent.Wait()
		container.CleanUp(containerID, volumes)
	}

	log.Infof("父进程运行结束")

	os.Exit(0)
}

func createDefaultBridge() {
	if !network.HasBridge(DEFAULT_BRIDGE) {
		//network create --driver bridge --subnet 192.168.10.1/24 testbridge
		cmd := exec.Command("/proc/self/exe", "network", "create", "--driver", "bridge", "--subnet", "192.168.10.1/24", DEFAULT_BRIDGE)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

func sendInitCommand(cmdArray []string, pipeWrite *os.File) (err error) {
	args := strings.Join(cmdArray, " ")
	log.Infof("发送初始化参数 %s", args)
	_, err = pipeWrite.WriteString(args)
	pipeWrite.Close()
	return
}
