package image

import (
	"encoding/json"
	"fmt"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	Domain                string      `json:"domain"`
	ImagePath             string      `json:"image_path"`
	Tag                   string      `json:"Tag"`
	ContentDigest         string      `json:"contentDigest"`
	suitableContentDigest string      `json:"suitableContentDigest"`
	Manifests             Manifests   `json:"manifests"`
	ImageLayerInfo        *LayersInfo `json:"image_layer_info"`
}

type Manifests struct {
	Manifests []struct {
		Digest    string `json:"digest"`
		MediaType string `json:"mediaType"`
		Platform  struct {
			Architecture string `json:"architecture"`
			Os           string `json:"os"`
		} `json:"platform"`
		Size int `json:"Size"`
	} `json:"manifests"`
	MediaType     string `json:"mediaType"`
	SchemaVersion int    `json:"schemaVersion"`
}

type LayersInfo struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"Size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"Size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

//1. Registry
//第一步 检查 Registry 是否有效
func (r *Registry) Check() error {
	//http://hub-mirror.c.163.com/v2/
	resp, err := http.Get(r.Domain + "/v2/")
	if err != nil {
		return fmt.Errorf("Docker Registry 地址无效 %s %v ", r.Domain, err)
	}
	defer resp.Body.Close()

	fmt.Printf("镜像仓库地址: %s \n", r.Domain)
	return nil
}

//2.1 Digest
//第二步 获取 Image 的摘要
func (r *Registry) Digest() error {
	//http://hub-mirror.c.163.com/v2/
	imageManifestsUrl := fmt.Sprintf("%s/v2/%s/manifests/%s", r.Domain, r.ImagePath, r.Tag)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", imageManifestsUrl, nil)
	req.Header.Add("Accept", `application/vnd.docker.distribution.manifest.v2+json`)
	req.Header.Add("Accept", `application/vnd.docker.distribution.manifest.v1+prettyjws`)
	req.Header.Add("Accept", `application/vnd.docker.distribution.manifest.list.v2+json`)
	req.Header.Add("Accept", `application/vnd.oci.image.index.v1+json`)
	req.Header.Add("Accept", `application/vnd.oci.image.manifest.v1+json`)
	imageManifestsResponse, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("Get Docker Manifests 失败 %s  %v", imageManifestsUrl, err)
	}

	defer imageManifestsResponse.Body.Close()

	//获取 Image 的 摘要
	if contentDigest, ok := imageManifestsResponse.Header["Docker-Content-Digest"]; ok {
		r.ContentDigest = contentDigest[0]
	} else {
		return fmt.Errorf("Get Docker Docker-Content-Digest 失败 ")
	}

	return r.digestByArchitecture("", "")
}

//2.2 Digest
//第三步 获取 Image 指定平台 的摘要
func (r *Registry) digestByArchitecture(architecture, os string) error {
	//获取当前系统平台
	if architecture == "" {
		architecture = runtime.GOARCH
	}

	imageManifestsUrl := fmt.Sprintf("%s/v2/%s/manifests/%s", r.Domain, r.ImagePath, r.ContentDigest)
	resp, err := http.Get(imageManifestsUrl)
	if err != nil {
		return fmt.Errorf("Docker Registry 地址无效 %s %v ", r.Domain, err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	manifests := &Manifests{}
	err = json.Unmarshal(body, manifests)
	if err != nil {
		return fmt.Errorf("Get Docker Manifests json.Unmarshal 失败 %s  %v", imageManifestsUrl, err)
	}
	for _, manifest := range manifests.Manifests {
		//平台和架构于当前一直的 Image
		if manifest.Platform.Architecture == architecture {
			r.suitableContentDigest = manifest.Digest

			fmt.Printf("镜像摘要: %s \n", manifest.Digest)
			return nil
		}

	}
	return fmt.Errorf("未找到合适的于当前系统镜像")
}

//3.1 Layer
//第四步 获取 Image 的 Layers
func (r *Registry) Layers() error {
	imageLayersUrl := fmt.Sprintf("%s/v2/%s/manifests/%s", r.Domain, r.ImagePath, r.suitableContentDigest)
	resp, err := http.Get(imageLayersUrl)
	if err != nil {
		return fmt.Errorf("获取 Image 的 Layers 失败 %s %v ", imageLayersUrl, err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	layersInfo := &LayersInfo{}
	err = json.Unmarshal(body, layersInfo)
	if err != nil {
		return fmt.Errorf("获取 Image 的 Layers json.Unmarshal 失败 %s  %v ", imageLayersUrl, err)
	}
	r.ImageLayerInfo = layersInfo
	return nil
}

//3.2 Layer
//第五步 下载所有的 Layers
func (r *Registry) DownloadLayers(layerDbPath string) error {

	/**
	layerdb/
	|-- mounts
	|   |-- 6746dbd115ed44f3e9c4cdc81d752e62240d24cbc278edf34f94347ff3f572e7
	|   |   |-- init-ID
	|   |   |-- mount-ID
	|   |   `-- parent
	|-- sha256
	|   |-- 1dad141bdb55cb5378a5cc3f4e81c10af14b74db9e861e505a3e4f81d99651bf
	|   |   |-- cache-ID
	|   |   |-- diff
	|   |   |-- Size
	|   |   `-- tar-split.json.gz
	*/
	layerDbSha256Path := path.Join(layerDbPath, "sha256")
	if _, err := os.Stat(layerDbSha256Path); os.IsNotExist(err) {
		os.MkdirAll(layerDbSha256Path, 0700)
	}

	fmt.Println("使用默认Tag: latest")
	fmt.Printf("latest: 从 %s 拉取\n", r.ImagePath)

	wait := sync.WaitGroup{}

	progress := &Progress{MaxLine: len(r.ImageLayerInfo.Layers)}
	for index, layer := range r.ImageLayerInfo.Layers {
		wait.Add(1)
		go func(index int, digest string) {
			defer wait.Done()

			layerDigest := strings.TrimPrefix(digest, "sha256:")
			fmt.Printf("\r%s: 拉取文件层", layerDigest[:13])

			downloadPath := path.Join(layerDbSha256Path, layerDigest)
			err := os.MkdirAll(downloadPath, 0700)
			if err != nil {
				log.Errorf("创建下载目录失败 %v", err)
				return
			}

			layerBlobsUrl := fmt.Sprintf("%s/v2/%s/blobs/%s", r.Domain, r.ImagePath, digest)

			filePath := path.Join(downloadPath, "tar-split.json.gz")

			if _, err := os.Stat(filePath); err == nil {
				//防止控制台刷新异常
				time.Sleep(100 * time.Millisecond)

				progress.skip(index, layerDigest)
				return
			}
			err = DownloadFile(index, layerDigest, filePath, layerBlobsUrl, progress)
			if err != nil {
				log.Errorf("DownloadFile error %v", err)
			}
		}(index, layer.Digest)
	}
	wait.Wait()

	return nil
}

//3.3 Image Layer
//第五步 下载镜像信息
func (r *Registry) DownloadImageContent(defaultImageContentPath string) error {

	/**
	|-- content
	|   `-- sha256
	|       |-- 05a68b853412e7487893459e565de4b119b8ebf99a15a598bce9dea0ce334411
	*/
	sha256Path := path.Join(defaultImageContentPath, "sha256")
	if _, err := os.Stat(sha256Path); os.IsNotExist(err) {
		os.MkdirAll(sha256Path, 0700)
	}

	err := os.MkdirAll(sha256Path, 0700)
	if err != nil {
		log.Errorf("创建下载目录失败 %v", err)
	}

	layerBlobsUrl := fmt.Sprintf("%s/v2/%s/blobs/%s", r.Domain, r.ImagePath, r.ImageLayerInfo.Config.Digest)

	digest := strings.TrimPrefix(r.ImageLayerInfo.Config.Digest, "sha256:")
	filePath := path.Join(sha256Path, digest)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = DownloadFile(0, digest, filePath, layerBlobsUrl, nil)
		if err != nil {
			log.Errorf("DownloadFile error %v", err)
		}
	}

	fmt.Printf("\n下载完成: %s\n", digest)

	return nil
}

func (r *Registry) ImageSize() int {
	size := 0
	for _, layer := range r.ImageLayerInfo.Layers {
		size += layer.Size
	}
	return size
}
