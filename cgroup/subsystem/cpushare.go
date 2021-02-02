package subsystem

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

type CpuSubSystem struct {
}

func (c *CpuSubSystem) Name() string {
	return "cpu"
}

func (c *CpuSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	log.Infof("设置 cgroup cpu share 开始，%s", res.CpuShare)
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, true); err == nil {
		if res.CpuShare == "" {
			log.Info("未配置 cgroup cpu share 跳过")
			return nil
		}
		if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "cpu.shares"), []byte(res.CpuShare), 0644); err != nil {
			log.Errorf("设置 cgroup cpu share 失败 %v", err)
			return fmt.Errorf("设置 cgroup cpu share 失败 %v", err)
		} else {
			log.Info("设置 cgroup cpu share 成功")
			return nil
		}
	} else {
		log.Errorf("设置 cgroup cpu share 失败 %v", err)
		return err
	}
}

func (c *CpuSubSystem) Apply(cgroupPath string, pid int, res *ResourceConfig) error {
	if res.CpuShare == "" {
		log.Info("未配置 cgroup cpu share 跳过")
		return nil
	}
	log.Infof("写入 cgroup cpu share pid=%d 开始", pid)
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, false); err == nil {
		if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			log.Errorf("写入 cgroup cpu share pid=%d 失败 %v", pid, err)
			return fmt.Errorf("写入 cgroup cpu share pid=%d 失败 %v", pid, err)
		} else {
			log.Infof("写入 cgroup cpu share pid=%d 成功", pid)
			return nil
		}
	} else {
		log.Errorf("写入 cgroup cpu share pid=%d 失败 %v", pid, err)
		return err
	}
}

func (c *CpuSubSystem) Remove(cgroupPath string) error {
	log.Infof("删除 cgroup cpu share 开始")
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, false); err == nil {
		return os.RemoveAll(cgroupAbsolutePath)
	} else {
		log.Errorf("删除 cgroup cpu share 失败 %v", err)
		return err
	}
}
