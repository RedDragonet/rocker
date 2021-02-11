package subsystem

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

type MemorySubSystem struct {
}

func (c *MemorySubSystem) Name() string {
	return "memory"
}

func (c *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	log.Debugf("设置 cgroup memory 开始，%s", res.MemoryLimit)
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, true); err == nil {
		if res.MemoryLimit == "" {
			log.Debugf("未配置 cgroup memory 跳过")
			return nil
		}
		if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {
			log.Errorf("设置 cgroup memory 失败 %v", err)
			return fmt.Errorf("设置 cgroup memory 失败 %v", err)
		} else {
			log.Debugf("设置 cgroup memory 成功")
			return nil
		}
	} else {
		log.Errorf("设置 cgroup memory 失败 %v", err)
		return err
	}
}

func (c *MemorySubSystem) Apply(cgroupPath string, pid int, res *ResourceConfig) error {
	if res.MemoryLimit == "" {
		log.Debugf("未配置 cgroup memory 跳过")
		return nil
	}
	log.Debugf("写入 cgroup memory pid=%d 开始", pid)
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, false); err == nil {
		if err := ioutil.WriteFile(path.Join(cgroupAbsolutePath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			log.Errorf("写入 cgroup memory pid=%d 失败 %v", pid, err)
			return fmt.Errorf("写入 cgroup memory pid=%d 失败 %v", pid, err)
		} else {
			log.Debugf("写入 cgroup memory pid=%d 成功", pid)
			return nil
		}
	} else {
		log.Errorf("写入 cgroup memory pid=%d 失败 %v", pid, err)
		return err
	}
}

func (c *MemorySubSystem) Remove(cgroupPath string) error {
	log.Debugf("删除 cgroup memory 开始")
	if cgroupAbsolutePath, err := GetCgroupPath(c.Name(), cgroupPath, false); err == nil {
		return os.RemoveAll(cgroupAbsolutePath)
	} else {
		log.Errorf("删除 cgroup memory 失败 %v", err)
		return err
	}
}
