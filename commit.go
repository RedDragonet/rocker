package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/RedDragonet/rocker/container"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

//镜像打包
/*

待打包 Layer
.
|-- busybox
    |-- diff
    |   |-- ******
    |-- lower
|-- 5d81f475ce28f0005821946a7e76d23dfcc28bb67263a9d5a8cb24126021feb8
    |-- diff
    |   |-- ******
    |-- lower

只打包 diff 目录

==========

打包后的目录

./
./busybox/
./busybox/layer.tar
./5d81f475ce28f0005821946a7e76d23dfcc28bb67263a9d5a8cb24126021feb8/
./5d81f475ce28f0005821946a7e76d23dfcc28bb67263a9d5a8cb24126021feb8/layer.tar
./repositories
./manifest.json

*/

func commit(id, output string) error {
	tmpCommitDir := path.Join("/tmp/rocker/commit/", id)
	defer func() {
		_ = os.RemoveAll(tmpCommitDir)
	}()

	log.Infof("容器打包镜像 %s", id)

	//遍历所有层的ID
	layers, err := getLayers(id)
	if err != nil {
		return err
	}

	layersJarFile := make([]string, len(layers))
	for index, layer := range layers {
		layerPath := path.Join(tmpCommitDir, layer)
		err := os.MkdirAll(layerPath, 0700)
		if err != nil {
			return err
		}

		jarFile := path.Join(layer, "layer.tar")
		layersJarFile[index] = jarFile
		//将每一层的diff目录，打包成  *ID*/layer.tar
		if _, err := exec.Command("tar", "-C", container.GetDiffPath(layer), "-cf", path.Join(tmpCommitDir, jarFile), ".").CombinedOutput(); err != nil {
			log.Errorf("容器打包镜像 layer error %s %v", layerPath, err)
			return fmt.Errorf("容器打包镜像 layer error %s %v", layerPath, err)
		}
	}

	err = createRepositoriesFile(tmpCommitDir, id, output)
	if err != nil {
		return err
	}

	err = createLayersFile(tmpCommitDir, layers)
	if err != nil {
		return err
	}

	//最终打包
	//所有的 jarFile file
	//******** 列出所有 files 打包符合 docker images 的规范
	/*
		[root@localhost ~]# tar -tf 3.tar
		busybox/layer.tar
		9701a03c637547cc19b465ea26323ee23071804e9cc8a03d38e2eacb9a5102fe/layer.tar
		repositories
		manifest.json


		******直接打包文件会出现
		[root@localhost ~]# tar -tf 3.tar
		./
		./busybox/layer.tar
		./9701a03c637547cc19b465ea26323ee23071804e9cc8a03d38e2eacb9a5102fe/layer.tar
		./repositories
		./manifest.json

		导致 tar 去查找文件，如manifest.json时，需要制定查找的文件为 ./manifest.json
	*/

	//直接打包目录会出现
	//
	files := append(layersJarFile, "repositories", "manifest.json")
	args := append([]string{"-C", tmpCommitDir, "-cf", output + ".tar"}, files...)

	if _, err := exec.Command("tar", args...).CombinedOutput(); err != nil {
		log.Errorf("容器打包镜像 error %s %s %s %v %v", id, output, err)
		return fmt.Errorf("容器打包镜像 error %s %s  %s %v %v", id, output, err)
	}
	return nil
}

func getLayers(id string) ([]string, error) {
	//lowers 层也同时打包
	lowerDirs, err := container.GetLowerDirs(id)
	if err != nil {
		return nil, err
	}

	layers := make([]string, len(lowerDirs)+1)
	for index, lowerDiffPath := range lowerDirs {
		layers[index] = path.Dir(lowerDiffPath)
	}
	//当前层在最后
	layers[len(layers)-1] = id
	return layers, nil
}

//创建当前容器描述文件
func createRepositoriesFile(dir, id, name string) error {
	//{"image_name":{"latest":"id"}}
	fileData := fmt.Sprintf(`{"%s":{"latest":"%s"}}`, name, id)
	err := ioutil.WriteFile(path.Join(dir, "repositories"), []byte(fileData), 0600)
	if err != nil {
		return err
	}

	return nil
}

//Layers
func createLayersFile(dir string, layers []string) error {
	//[{"Layers":["12sbv","qe123","123sdf"]}]
	layersJson, err := json.Marshal(layers)
	if err != nil {
		return err
	}

	fileData := fmt.Sprintf(`[{"Layers":%s}]`, string(layersJson))
	err = ioutil.WriteFile(path.Join(dir, "manifest.json"), []byte(fileData), 0600)
	if err != nil {
		return err
	}

	return nil
}
