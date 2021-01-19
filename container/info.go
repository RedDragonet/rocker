package container

import "time"

//https://github.com/moby/moby/blob/46cdcd206c56172b95ba5c77b827a722dab426c5/container/state.go#L17
type ContainerInfo struct {
	//容器状态
	State   State         `json:"State"`
	ID      string        `json:"ID"`
	Created time.Time     `json:"Created"`
	Args    []interface{} `json:"Args"`
	Config  Config        `json:"Config"`
	Name    string        `json:"Name"`
}

type State struct {
	Running           bool      `json:"Running"`
	Paused            bool      `json:"Paused"`
	Restarting        bool      `json:"Restarting"`
	OOMKilled         bool      `json:"OOMKilled"`
	RemovalInProgress bool      `json:"RemovalInProgress"`
	Dead              bool      `json:"Dead"`
	Pid               int       `json:"Pid"`
	ExitCode          int       `json:"ExitCode"`
	StartedAt         time.Time `json:"StartedAt"`
	FinishedAt        time.Time `json:"FinishedAt"`
}

type Config struct {
	Cmd     []string `json:"Cmd"`
	Image   string   `json:"Image"`
	Volumes string   `json:"Volumes"`
}

func (s *State) String() string {
	if s.Running {
		if s.Paused {
			return "Paused"
		}
		if s.Restarting {
			return "Restarting"
		}
		return "Up"
	}

	if s.RemovalInProgress {
		return "Removal In Progress"
	}

	if s.Dead {
		return "Dead"
	}

	if s.StartedAt.IsZero() {
		return "Created"
	}

	if s.FinishedAt.IsZero() {
		return ""
	}

	return "Exited"
}
