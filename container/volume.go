package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

func MountVolumeSlice(rootfs string, volumeSlice []string) error {
	for _, volume := range volumeSlice {
		err := mountVolume(rootfs, volume)
		if err != nil {
			return err
		}
	}
	return nil
}

func mountVolume(rootfs, volume string) error {
	source, target, err := volumeSplit(rootfs, volume)
	if err != nil {
		return err
	}

	//不存在则创建目录
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if sourceFile, err := os.Stat(source); !os.IsNotExist(err) {
			targetDir := target
			if !sourceFile.IsDir() {
				targetDir = path.Dir(target)
			}
			log.Infof("mount volume create targetDir %s done.", targetDir)
			os.MkdirAll(targetDir, 0700)
		}
	}

	if err := syscall.Mount(source, target, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		log.Errorf("mount volume %s => %s error ", source, target, err)
		return err
	}
	log.Infof("mount volume %s done. %v", volume, err)
	return nil
}
func volumeSplit(rootfs, volume string) (source, target string, err error) {
	if volume == "" {
		return
	}

	volumeSplit := strings.Split(volume, ":")
	if len(volumeSplit) != 2 {
		err = fmt.Errorf("错误的Volume参数 %s", volume)
		return
	}
	source = volumeSplit[0]
	target = path.Join(rootfs, volumeSplit[1])
	return
}

func UnMountVolumeSlice(id string, volumeSlice []string) error {
	if len(volumeSlice) == 0 {
		return nil
	}

	for _, volume := range volumeSlice {
		err := unMountVolume(id, volume)
		if err != nil {
			return err
		}
	}
	return nil
}

func unMountVolume(id, volume string) error {
	if volume == "" {
		return nil
	}

	//TODO::判断是否创建了容器内的目录，删除

	_, target, err := volumeSplit(getMergedPath(id), volume)
	if err != nil {
		return err
	}
	if _, err := exec.Command("umount", target).CombinedOutput(); err != nil {
		log.Errorf("umount volume %s failed. %v", target, err)
		return err
	}

	log.Info("umount volume %s done. %v", volume, err)
	return nil
}
