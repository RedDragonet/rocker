package subsystem

type ResourceConfig struct {
	MemoryLimit string
	CpuShare    string
	CpuSet      string
}

type Subsystem interface {
	Name() string
	Set(path string, res *ResourceConfig) error
	Apply(path string, pid int, res *ResourceConfig) error
	Remove(path string) error
}

var (
	SubsystemIns = []Subsystem{
		&CpuSetSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)
