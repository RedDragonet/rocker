package container

import (
	"fmt"
	image2 "github.com/RedDragonet/rocker/image"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/RedDragonet/rocker/pkg/pidlog"
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
		_ = os.MkdirAll(home, 0755)
	}
}

//目前暂定rootFs是当前目录下的tar包
//return 挂载的 mntUrl
func NewWorkSpace(image, id string) (string, error) {
	//overlayFS mount
	//支持将打包的容器 恢复
	i := image2.Get(image)
	if i == nil {
		//pull image
		cmd := exec.Command("/proc/self/exe", "pull", image)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()

		image2.Init()
		i = image2.Get(image)
	}

	if i == nil {
		return "", fmt.Errorf("镜像获取失败")
	}

	//rootfs := strings.Split(rootfsPath, ".")[0]
	layers, err := getLayersTarFile(i)
	if err != nil {
		return "", err
	}

	//从底层往上，创建每一层
	err = LoopExtract(layers, i)
	if err != nil {
		return "", err
	}

	//创建容器层
	err = create(id, layers[len(layers)-1], nil)
	if err != nil {
		return "", err
	}

	mntUrl, err := get(id)
	return mntUrl, err
}

//循环解压image tar包
func LoopExtract(layers []string, i *image2.Image) error {
	var err error
	for index, layer := range layers {
		log.Debugf("Extract Layer %s", layer)

		layer = strings.TrimPrefix(layer, "sha256:")
		if index > 0 {
			//上一级的layer
			err = create(layer, layers[index-1], i)
		} else {
			err = create(layer, "", i)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

//删除当前container层的目录
func DelWorkSpace(id string) error {
	dir := path.Join(home, id)

	mergedDirPath := path.Join(dir, mergedDirName)

	//if _, err := exec.Command("umount", path.Join(mergedDirPath, "/proc")).CombinedOutput(); err != nil {
	//	log.Errorf("umount  /proc failed. %v", err)
	//}

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

func create(id, parent string, i *image2.Image) error {
	log.Debugf("【overflowFS】创建目录 %s %s", id, parent)

	var retErr error
	dir := path.Join(home, id)

	if exist, _ := os.Stat(dir); exist != nil {
		log.Debugf("【overflowFS】目录存在跳过 %s %s", id, parent)
		return nil
	}
	//失败时
	defer func() {
		if retErr != nil {
			log.Errorf("【overflowFS】失败 %s %s", id, parent)
			_ = os.RemoveAll(dir)
		}
	}()

	retErr = os.Mkdir(dir, 0755)
	if retErr != nil {
		return retErr
	}

	diffDirPath := path.Join(dir, diffDirName)
	workDirPath := path.Join(dir, workDirName)
	mergedDirPath := path.Join(dir, mergedDirName)

	retErr = os.Mkdir(diffDirPath, 0755)
	if retErr != nil {
		log.Errorf("【overflowFS】创建diff目录失败 %s %s %s %v", id, parent, diffDirPath, retErr)
		return retErr
	}

	retErr = os.Mkdir(workDirPath, 0755)
	if retErr != nil {
		log.Errorf("【overflowFS】创建work目录失败 %s %%s %s %v", id, parent, workDirPath, retErr)
		return retErr
	}

	retErr = os.Mkdir(mergedDirPath, 0755)
	if retErr != nil {
		log.Errorf("【overflowFS】创建merge目录失败 %s %s %s %v", id, parent, mergedDirPath, retErr)
		return retErr
	}

	//非容器层
	if i != nil {
		//解压对应层的文件
		retErr = ApplyDiff(id, i)
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
func ApplyDiff(id string, i *image2.Image) error {
	var retErr error
	//home := GetHome()
	dirPath := dirPath(id)
	diffDirPath := getDiffPath(id)
	log.Debugf("解压 layer.tar %s %s %s 开始", id, i.Repository, dirPath)

	// 通过 Pull 方式
	if _, retErr = exec.Command("tar", "-zxf", path.Join(i.LayerPath(id), "tar-split.json.gz"), "-C", diffDirPath).CombinedOutput(); retErr != nil {
		log.Errorf("tar", "-zxf", path.Join(i.LayerPath(id), "tar-split.json.gz"), "-C", diffDirPath)
		return retErr
	}

	log.Debugf("解压 layer.tar %s => %s 成功", path.Join(dirPath, "layer.tar"), diffDirPath)

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
func GetLowerDirs(id string) ([]string, error) {
	var lowersArray []string
	lowers, err := ioutil.ReadFile(path.Join(dirPath(id), lowerFile))
	if err == nil {
		for _, s := range strings.Split(string(lowers), ":") {
			lowersArray = append(lowersArray, s)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return lowersArray, nil
}

func GetHome() string {
	return home
}

func GetDiffDirName() string {
	return diffDirName
}

func GetDiffPath(id string) string {
	return getDiffPath(id)
}

//解析当前容器描述文件
func getLayersTarFile(i *image2.Image) ([]string, error) {
	if i.ImageLayerInfo == nil {
		return nil, fmt.Errorf("ImageLayerInfo is nil")
	}

	layers := make([]string, len(i.ImageLayerInfo.Layers))
	for k, v := range i.ImageLayerInfo.Layers {
		layers[k] = strings.TrimPrefix(v.Digest, "sha256:")
	}

	reverse(layers)

	return layers, nil
}
func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}
