package subsystem

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"

	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

//查找对应 subsystem 的挂载点
func FindCgroupMountPoint(subsystem string) (string, error) {
	log.Infof("查找对应 subsystem 的挂载点 %s 开始", subsystem)

	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		log.Errorf("查找对应 subsystem 的挂载点 %s 打开 /proc/self/mountinfo 失败 %v", subsystem, err)
		return "", fmt.Errorf("查找对应 subsystem 的挂载点 %s 打开 /proc/self/mountinfo 失败 %v", subsystem, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		//
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				log.Infof("查找对应 subsystem 的挂载点 %s 成功 %s", subsystem, fields[4])

				return fields[4], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("查找对应 subsystem 的挂载点 %s 失败 %v", subsystem, err)
		return "", fmt.Errorf("查找对应 subsystem 的挂载点 %s 失败 %v", subsystem, err)
	}

	log.Errorf("查找对应 subsystem 的挂载点 %s 失败，未找到对于挂载点", subsystem, err)
	return "", fmt.Errorf("查找对应 subsystem 的挂载点 %s 失败，未找到对于挂载点 %v", subsystem, err)
}

//获取/创建 cgroup 对应 subsystem 的目录
func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot, err := FindCgroupMountPoint(subsystem)
	if err != nil {
		return "", err
	}
	cgroupAbsolutePath := path.Join(cgroupRoot, cgroupPath)

	log.Infof("获取/创建 cgroup 对应 subsystem 的目录 %s", cgroupAbsolutePath)
	if _, err := os.Stat(cgroupAbsolutePath); err != nil {
		//当目录不存时，并且希望自动创建时
		if os.IsNotExist(err) && autoCreate {
			//创建目录
			if err := os.Mkdir(cgroupAbsolutePath, 0755); err == nil {
				log.Infof("获取/创建 cgroup 对应 subsystem 的目录 成功", cgroupAbsolutePath)
				return cgroupAbsolutePath, nil
			} else {
				log.Errorf("获取/创建 cgroup 对应 subsystem 的目录 %s，失败 %v", cgroupAbsolutePath, err)
				return "", fmt.Errorf("获取/创建 cgroup 对应 subsystem 的目录 %s，失败 %v", cgroupAbsolutePath, err)
			}
		} else {
			log.Errorf("获取/创建 cgroup 对应 subsystem 的目录 %s，失败 %v", cgroupAbsolutePath, err)
			return "", err
		}
	}
	log.Infof("获取/创建 cgroup 对应 subsystem 的目录 已存在", cgroupAbsolutePath)
	return cgroupAbsolutePath, nil
}
