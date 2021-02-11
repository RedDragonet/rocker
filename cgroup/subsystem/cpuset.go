package subsystem

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

type CpuSetSubSystem struct {
}

func (c *CpuSetSubSystem) Name() string {
	return "cpuset"
}

func (c *CpuSetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	log.Infof("设置 cgroup cpu set 开始, %s,%b", res.CpuSet)
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, true); err == nil {
		if res.CpuSet == "" {
			log.Debugf("未配置 cgroup cpu set 跳过")
			return nil
		}
		if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "cpuset.cpus"), []byte(res.CpuSet), 0644); err != nil {
			log.Errorf("设置 cgroup cpu set 失败 %v", err)
			return fmt.Errorf("设置 cgroup cpu set 失败 %v", err)
		} else {
			//fix:同时需要设置 cpuset.mems
			if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "cpuset.mems"), []byte("0"), 0644); err != nil {
				log.Errorf("设置 cgroup cpu set cpuset.mems 失败 %v", err)
			}
			log.Debugf("设置 cgroup cpu set 成功")
			return nil
		}
	} else {
		log.Errorf("设置 cgroup cpu set 失败 %v", err)
		return err
	}
}

func (c *CpuSetSubSystem) Apply(cgroupPath string, pid int, res *ResourceConfig) error {
	if res.CpuSet == "" {
		log.Debugf("未配置 cgroup cpu set 跳过")
		return nil
	}
	log.Infof("写入 cgroup cpu set pid=%d 开始", pid)
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, false); err == nil {
		if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			log.Errorf("写入 cgroup cpu set pid=%d 失败 %v", pid, err)
			return fmt.Errorf("写入 cgroup cpu set pid=%d 失败 %v", pid, err)
		} else {
			log.Debugf("写入 cgroup cpu set pid=%d 成功", pid)
			return nil
		}
	} else {
		log.Errorf("写入 cgroup cpu set pid=%d 失败 %v", pid, err)
		return err
	}
}

func (c *CpuSetSubSystem) Remove(cgroupPath string) error {
	log.Infof("删除 cgroup cpu set 开始")
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, false); err == nil {
		return os.RemoveAll(cgroupAbsolutePath)
	} else {
		log.Errorf("删除 cgroup cpu set 失败 %v", err)
		return err
	}
}
