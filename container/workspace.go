package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

//创建OverlayFS
//参考 https://github.com/moby/moby/blob/b5f863c67e6ffbaadedb94c885c3c50c625e4eb8/daemon/graphdriver/fuse-overlayfs/fuseoverlayfs.go#L172
const (
	home          = "/var/lib/rocker/overlay2"
	diffDirName   = "diff"
	workDirName   = "work"
	mergedDirName = "merged"
	lowerFile     = "lower"
)

func init() {
	//创建 home 文件夹
	if _, err := os.Stat(home); os.IsNotExist(err) {
		_ = os.MkdirAll(home, 0700)
	}
}



//目前暂定rootFs是当前目录下的tar包
func NewWorkSpace(rootfsPath, id string) (mntUrl string, err error) {
	//overlayFS mount

	rootfs := strings.Split(rootfsPath, ".")[0]
	err = create(rootfs, "", rootfsPath)
	if err != nil {
		return
	}

	err = create(id, rootfs, "")
	if err != nil {
		return
	}

	mntUrl, err = get(id)
	return
}

//删除当前container层的目录
func DelWorkSpace(id string) error {
	dir := path.Join(home, id)

	mergedDirPath := path.Join(dir, mergedDirName)
	if _, err := exec.Command("umount", mergedDirPath).CombinedOutput(); err != nil {
		log.Errorf("umount overlayFS %s failed. %v", mergedDirPath, err)
		return err
	}
	log.Infof("umount overlayFS %s done.", mergedDirPath)
	log.Infof("RemoveAll %s start.", dir)
	return os.RemoveAll(dir)
}

func get(id string) (string, error) {
	dir := dirPath(id)
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}

	diffDir := path.Join(dir, diffDirName)
	workDir := path.Join(dir, workDirName)
	mergeDir := path.Join(dir, mergedDirName)
	lowers, err := ioutil.ReadFile(path.Join(dir, lowerFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("暂时不支持")
		}
		return "", err
	}

	//将相对路径转换为绝对路径 /XXXXX => /var/lib/rocker/overlay2/XXXXX
	splitLowers := strings.Split(string(lowers), ":")
	absLowers := make([]string, len(splitLowers))
	for i, s := range splitLowers {
		absLowers[i] = path.Join(home, s)
	}
	opts := "lowerdir=" + strings.Join(absLowers, ":") + ",upperdir=" + diffDir + ",workdir=" + workDir
	_, err = exec.Command("mount", "-t", "overlay", "-o", opts, "none", mergeDir).CombinedOutput()
	if err != nil {
		log.Errorf("Mount overlayFS %s => %s failed. %v", opts, mergeDir, err)
		return "", err
	}
	log.Infof("Mount overlayFS %s => %s done", opts, mergeDir)
	return mergeDir, nil
}

func create(id, parent, diffTar string) error {
	log.Infof("【overflowFS】创建目录 %s %s %s", id, parent, diffTar)

	var retErr error
	dir := path.Join(home, id)

	if exist, _ := os.Stat(dir); exist != nil {
		log.Infof("【overflowFS】目录存在跳过 %s %s %s", id, parent, diffTar)
		return nil
	}
	//失败时
	defer func() {
		if retErr != nil {
			log.Infof("【overflowFS】失败 %s %s %s", id, parent, diffTar)
			_ = os.RemoveAll(dir)
		}
	}()

	retErr = os.Mkdir(dir, 0700)
	if retErr != nil {
		return retErr
	}

	diffDirPath := path.Join(dir, diffDirName)
	workDirPath := path.Join(dir, workDirName)
	mergedDirPath := path.Join(dir, mergedDirName)

	retErr = os.Mkdir(diffDirPath, 0700)
	if retErr != nil {
		log.Errorf("【overflowFS】创建diff目录失败 %s %s %s %s %v", id, parent, diffTar, diffDirPath, retErr)
		return retErr
	}

	retErr = os.Mkdir(workDirPath, 0700)
	if retErr != nil {
		log.Errorf("【overflowFS】创建work目录失败 %s %s %s %s %v", id, parent, diffTar, workDirPath, retErr)
		return retErr
	}

	retErr = os.Mkdir(mergedDirPath, 0700)
	if retErr != nil {
		log.Errorf("【overflowFS】创建merge目录失败 %s %s %s %s %v", id, parent, diffTar, mergedDirPath, retErr)
		return retErr
	}

	//解压对应层的文件
	if diffTar != "" {
		retErr = ApplyDiff(id, diffTar)
		if retErr != nil {
			return retErr
		}
	}

	//是否存在上一层
	if parent != "" {
		lower, err := getLower(parent)
		if err != nil {
			retErr = err
			return retErr
		}
		if lower != "" {
			if err := ioutil.WriteFile(path.Join(dir, lowerFile), []byte(lower), 0666); err != nil {
				return err
			}
		}
	}
	return nil
}

//将每层的文件压缩比解压到对应的Diff文件夹
func ApplyDiff(id, diffTar string) error {
	var retErr error
	diffDirPath := getDiffPath(id)
	log.Infof("解压RootFS %s => %s 开始", diffTar, diffDirPath)
	if _, retErr = os.Stat(diffTar); retErr == nil {
		if _, retErr = exec.Command("tar", "-xvf", diffTar, "-C", diffDirPath).CombinedOutput(); retErr != nil {
			log.Errorf("解压RootFS %s => %s 失败 %v", diffTar, diffDirPath, retErr)
		}
		log.Infof("解压RootFS %s => %s 成功", diffTar, diffDirPath)
	}
	return retErr
}

func dirPath(id string) string {
	return path.Join(home, id)
}

func getDiffPath(id string) string {
	dir := dirPath(id)
	return path.Join(dir, diffDirName)
}

func getMergedPath(id string) string {
	dir := dirPath(id)
	return path.Join(dir, mergedDirName)
}

func getLower(parent string) (string, error) {
	lowers, err := ioutil.ReadFile(path.Join(dirPath(parent), lowerFile))
	if err != nil {
		if os.IsNotExist(err) {
			return path.Join(parent, diffDirName), nil
		} else {
			return "", err
		}
	}
	lowersArray := append(strings.Split(string(lowers), ":"), path.Join(parent, diffDirName))
	return strings.Join(lowersArray, ":"), nil
}

//获得所有的lowerDir
//func getLowerDirs(id string) ([]string, error) {
//	var lowersArray []string
//	lowers, err := ioutil.ReadFile(path.Join(dirPath(id), lowerFile))
//	if err == nil {
//		for _, s := range strings.Split(string(lowers), ":") {
//			lowersArray = append(lowersArray, path.Join(home, s))
//		}
//	} else if !os.IsNotExist(err) {
//		return nil, err
//	}
//	return lowersArray, nil
//}
