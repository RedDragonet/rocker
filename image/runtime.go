package image

import "time"

//镜像运行配置
type Runtime struct {
	Architecture string `json:"architecture"`
	Config       struct {
		Hostname     string `json:"Hostname"`
		Domainname   string `json:"Domainname"`
		User         string `json:"User"`
		AttachStdin  bool   `json:"AttachStdin"`
		AttachStdout bool   `json:"AttachStdout"`
		AttachStderr bool   `json:"AttachStderr"`
		ExposedPorts struct {
			Eight0TCP struct {
			} `json:"80/tcp"`
		} `json:"ExposedPorts"`
		Tty        bool        `json:"Tty"`
		OpenStdin  bool        `json:"OpenStdin"`
		StdinOnce  bool        `json:"StdinOnce"`
		Env        []string    `json:"Env"`
		Cmd        []string    `json:"Cmd"`
		Image      string      `json:"Image"`
		Volumes    interface{} `json:"Volumes"`
		WorkingDir string      `json:"WorkingDir"`
		Entrypoint []string    `json:"Entrypoint"`
		OnBuild    interface{} `json:"OnBuild"`
		Labels     struct {
			Maintainer string `json:"maintainer"`
		} `json:"Labels"`
		StopSignal string `json:"StopSignal"`
	} `json:"config"`
	Container       string `json:"container"`
	ContainerConfig struct {
		Hostname     string `json:"Hostname"`
		Domainname   string `json:"Domainname"`
		User         string `json:"User"`
		AttachStdin  bool   `json:"AttachStdin"`
		AttachStdout bool   `json:"AttachStdout"`
		AttachStderr bool   `json:"AttachStderr"`
		ExposedPorts struct {
			Eight0TCP struct {
			} `json:"80/tcp"`
		} `json:"ExposedPorts"`
		Tty        bool        `json:"Tty"`
		OpenStdin  bool        `json:"OpenStdin"`
		StdinOnce  bool        `json:"StdinOnce"`
		Env        []string    `json:"Env"`
		Cmd        []string    `json:"Cmd"`
		Image      string      `json:"Image"`
		Volumes    interface{} `json:"Volumes"`
		WorkingDir string      `json:"WorkingDir"`
		Entrypoint []string    `json:"Entrypoint"`
		OnBuild    interface{} `json:"OnBuild"`
		Labels     struct {
			Maintainer string `json:"maintainer"`
		} `json:"Labels"`
		StopSignal string `json:"StopSignal"`
	} `json:"container_config"`
	Created       time.Time `json:"created"`
	DockerVersion string    `json:"docker_version"`
	History       []struct {
		Created    time.Time `json:"created"`
		CreatedBy  string    `json:"created_by"`
		EmptyLayer bool      `json:"empty_layer,omitempty"`
	} `json:"history"`
	Os     string `json:"os"`
	Rootfs struct {
		Type    string   `json:"type"`
		DiffIds []string `json:"diff_ids"`
	} `json:"rootfs"`
}
