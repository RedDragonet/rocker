package cgroup

import (
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

type CgroupManager struct {
	// cgroup在hierarchy中的路径 相当于创建的cgroup目录相对于root cgroup目录的路径
	Path string
	// 资源配置
	Resource *subsystem.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

// 将进程pid加入到这个cgroup中
func (c *CgroupManager) Apply(pid int, res *subsystem.ResourceConfig) error {
	for _, subSysIns := range subsystem.SubsystemIns {
		err := subSysIns.Apply(c.Path, pid, res)
		if err != nil {
			return err
		}
	}
	return nil
}

// 设置cgroup资源限制
func (c *CgroupManager) Set(res *subsystem.ResourceConfig) error {
	for _, subSysIns := range subsystem.SubsystemIns {
		err := subSysIns.Set(c.Path, res)
		if err != nil {
			return err
		}
	}
	return nil
}

//释放cgroup
func (c *CgroupManager) Destroy() error {
	for _, subSysIns := range subsystem.SubsystemIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			log.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}
