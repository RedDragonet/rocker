package main

import (
	"fmt"
	"github.com/RedDragonet/rocker/container"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"path"
)

//镜像打包
/*
[root@localhost f97948b47fd29c0ded25a9da85ac368b911117ada31ed21839ecb6d8dda2fb58]# tree
.
|-- diff
|   `-- test4
|-- lower
|-- merged
|   |-- bin


只打包 diff 目录，和 lower 文件
*/

func commit(id, output string) error {
	home := container.GetHome()
	diffDirName := container.GetDiffDirName()
	lowerFile := container.GetLowerFile()

	log.Infof("容器打包镜像 %s", id)

	//打包当前层
	tarPathList := []string{path.Join(id, diffDirName), path.Join(id, lowerFile)}

	//lowers 层也同时打包
	lowerDirs, err := container.GetLowerDirs(id)
	if err != nil {
		return err
	}

	for index, lowerDiffPath := range lowerDirs {
		tarPathList = append(tarPathList, lowerDiffPath)

		//最后一层没有 Lower file
		if index+1 != len(lowerDirs) {
			tarPathList = append(tarPathList, path.Join(path.Dir(lowerDiffPath), lowerFile))
		}
	}

	//参数拼装
	ops := append([]string{"-C", home, "-cf", output + ".tar"}, tarPathList...)
	if _, err := exec.Command("tar", ops...).CombinedOutput(); err != nil {
		log.Errorf("容器打包镜像 error %s %s %s %v %v", id, output, ops, err)
		return fmt.Errorf("容器打包镜像 error %s %s  %s %v %v", id, output, ops, err)
	}

	return nil
}
