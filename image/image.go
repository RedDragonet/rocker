package image

import (
	"encoding/json"
	"fmt"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

var (
	defaultImagePath        = "/var/lib/rocker/image/overlay2"
	imageDb                 = "imagedb"
	layerDb                 = "layerdb"
	repositoriesPath        = defaultImagePath + "/repositories/"
	defaultImageContentPath = defaultImagePath + "/" + imageDb + "/content"

	///var/lib/docker/image/overlay2/imagedb/content/sha256

	//中科大镜像
	registryBase = "https://ustc-edu-cn.mirror.aliyuncs.com"
	images       = map[string]*Image{}
)

//REPOSITORY        TAG       IMAGE ID       CREATED         SIZE
type Image struct {
	Repository string `json:"Repository"`
	Tag        string `json:"Tag"`
	//digest     string `json:"digest"`
	ID             string      `json:"ID"`
	Created        time.Time   `json:"Created"`
	Size           int         `json:"Size"`
	ImageLayerInfo *LayersInfo `json:"image_layer_info"`
	Runtime        *Runtime    `json:"runtime"`
}

func (image *Image) LayerPath(layerId string) string {
	layerPath := path.Join(path.Join(defaultImagePath, layerDb))
	sha256Path := path.Join(path.Join(layerPath, "sha256"))
	return path.Join(sha256Path, layerId)
}

func (image *Image) GetRuntime() (*Runtime, error) {
	if image.Runtime != nil {
		return image.Runtime, nil
	}

	manifestFile := image.manifestFile()
	file, err := os.Open(manifestFile)
	if err != nil {
		log.Errorf("error：", err)
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("error：", err)
		return nil, err
	}

	r := &Runtime{}
	image.Runtime = r
	return r, json.Unmarshal(data, r)
}

func (image *Image) manifestFile() string {
	sha256Path := path.Join(path.Join(defaultImageContentPath, "sha256"))
	return path.Join(sha256Path, image.ID)
}

func (image *Image) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}

	imagPath := path.Join(dumpPath, image.ID)
	nwFile, err := os.OpenFile(imagPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("error：", err)
		return err
	}
	defer nwFile.Close()

	nwJson, err := json.Marshal(image)

	if err != nil {
		log.Errorf("error：", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		log.Errorf("error：", err)
		return err
	}
	return nil
}

func (image *Image) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, image.ID)); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		fmt.Println("sc", path.Join(dumpPath, image.ID))
		return os.Remove(path.Join(dumpPath, image.ID))
	}
}

func (image *Image) load(dumpPath string) error {
	imageConfigFile, err := os.Open(dumpPath)
	defer imageConfigFile.Close()
	if err != nil {
		return err
	}
	imageJson := make([]byte, 2000)
	n, err := imageConfigFile.Read(imageJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(imageJson[:n], image)
	if err != nil {
		log.Errorf("Error load image info", err)
		return err
	}
	return nil
}

//遍历所有已经配置过的IMAGE
func Init() error {
	if _, err := os.Stat(repositoriesPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(repositoriesPath, 0644)
		} else {
			return err
		}
	}

	filepath.Walk(repositoriesPath, func(imagPath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(imagPath, "/") {
			return nil
		}
		_, imageId := path.Split(imagPath)
		image := &Image{
			ID: imageId,
		}

		if err := image.load(imagPath); err != nil {
			log.Errorf("遍历 Image 失败: %s %s %v", imagPath, imageId, err)
		}

		images[image.Repository] = image
		return nil
	})

	return nil
}

func ListImage() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "REPOSITORY\tTAG\tIMAGE ID\tCREATED\tSIZE\n")
	for _, image := range images {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			image.Repository,
			image.Tag,
			image.ID[:13],
			image.Created.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%s", humanSize(uint64(image.Size))),
		)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}

func Get(imageName string) *Image {
	if image, ok := images[fixImageName(imageName)]; ok {
		return image
	}
	return nil
}

func fixImageName(imageName string) string {
	imageName = strings.Trim(imageName, "/")
	imageNameSplit := strings.Split(imageName, "/")
	if len(imageNameSplit) < 2 {
		imageName = "library/" + imageNameSplit[0]
	}
	return imageName
}

func Delete(imageName string) error {
	image, ok := images[fixImageName(imageName)]
	if !ok {
		return fmt.Errorf("未找到对应的 Image: %s", imageName)
	}

	return image.remove(repositoriesPath)
}

func Pull(imageName string) error {
	domain, imagePath, err := parseImageUrl(imageName)
	if err != nil {
		return err
	}

	registry := &Registry{
		Domain:    domain,
		ImagePath: imagePath,
		Tag:       "latest",
	}
	//1. 判断 mirror 是否有效
	err = registry.Check()
	if err != nil {
		return err
	}

	//2. 获取镜像信息
	//http://hub-mirror.c.163.com/v2/library/nginx/manifests/latest
	err = registry.Digest()
	if err != nil {
		return err
	}

	//3. 获取镜像层信息
	err = registry.Layers()
	if err != nil {
		return err
	}

	//4. 下载镜像层信息
	err = registry.DownloadLayers(path.Join(defaultImagePath, layerDb))
	if err != nil {
		return err
	}

	//5. 写入镜像信息
	err = registry.DownloadImageContent(defaultImageContentPath)
	if err != nil {
		return err
	}

	image := &Image{
		Repository:     imagePath,
		Tag:            "latest",
		ID:             strings.TrimPrefix(registry.ImageLayerInfo.Config.Digest, "sha256:"),
		Created:        time.Now(),
		Size:           registry.ImageSize(),
		ImageLayerInfo: registry.ImageLayerInfo,
	}

	err = image.dump(repositoriesPath)
	if err != nil {
		return err
	}

	images[image.Repository] = image
	return nil
}

func parseImageUrl(imageName string) (string, string, error) {
	imageRegistryBase := registryBase

	if strings.HasPrefix(imageName, "http") {
		imageUrl, err := url.Parse(imageName)
		if err != nil {
			log.Errorf("image error parse: %s", imageName, err)
			return "", "", err
		}
		imageRegistryBase = imageUrl.Scheme + "://" + imageUrl.Host
		imageName = imageUrl.Path
	}
	imageName = strings.Trim(imageName, "/")
	imageNameSplit := strings.Split(imageName, "/")
	if len(imageNameSplit) < 2 {
		imageName = "library/" + imageNameSplit[0]
	}

	return imageRegistryBase, imageName, nil
}
