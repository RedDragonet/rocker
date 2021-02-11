package container

import (
	"errors"
	"fmt"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

type Type rune

const (
	Wildcard = -1
)

const (
	WildcardDevice Type = 'a'
	BlockDevice    Type = 'b'
	CharDevice     Type = 'c' // or 'u'
	FifoDevice     Type = 'p'
)

type Permissions string

type Rule struct {
	// Type of device ('c' for char, 'b' for block). If set to 'a', this rule
	// acts as a wildcard and all fields other than Allow are ignored.
	Type Type `json:"type"`

	// Major is the device's major number.
	Major int64 `json:"major"`

	// Minor is the device's minor number.
	Minor int64 `json:"minor"`

	// Permissions is the set of permissions that this rule applies to (in the
	// cgroupv1 format -- any combination of "rwm").
	Permissions Permissions `json:"permissions"`

	// Allow specifies whether this rule is allowed.
	Allow bool `json:"allow"`
}

func (d *Rule) Mkdev() (uint64, error) {
	if d.Major == Wildcard || d.Minor == Wildcard {
		return 0, errors.New("cannot mkdev() device with wildcards")
	}
	return unix.Mkdev(uint32(d.Major), uint32(d.Minor)), nil
}

type Device struct {
	Rule

	// Path to the device.
	Path string `json:"path"`

	// FileMode permission bits for the device.
	FileMode os.FileMode `json:"file_mode"`

	// Uid of the device.
	Uid uint32 `json:"uid"`

	// Gid of the device.
	Gid uint32 `json:"gid"`
}

//系统启动创建默认设备
func createDefaultDevice(rootfs string) error {
	oldMask := unix.Umask(0000)
	for _, node := range AllowedDevices {
		// containers running in a user namespace are not allowed to mknod
		// devices so we can just bind mount it from the host.
		if err := createDeviceNode(rootfs, node, true); err != nil {
			unix.Umask(oldMask)
			log.Errorf("unix.Umask error %v", err)
			return err
		}
	}
	unix.Umask(oldMask)

	return setupDevSymlinks(rootfs)
}

func setupDevSymlinks(rootfs string) error {
	var links = [][2]string{
		{"/proc/self/fd", "/dev/fd"},
		{"/proc/self/fd/0", "/dev/stdin"},
		{"/proc/self/fd/1", "/dev/stdout"},
		{"/proc/self/fd/2", "/dev/stderr"},
	}
	// kcore support can be toggled with CONFIG_PROC_KCORE; only create a symlink
	// in /dev if it exists in /proc.
	if _, err := os.Stat("/proc/kcore"); err == nil {
		links = append(links, [2]string{"/proc/kcore", "/dev/core"})
	}
	for _, link := range links {
		var (
			src = link[0]
			dst = filepath.Join(rootfs, link[1])
		)
		if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
			return fmt.Errorf("symlink %s %s %s", src, dst, err)
		}
	}
	return nil
}

func createDeviceNode(rootfs string, node *Device, bind bool) error {
	if node.Path == "" {
		// The node only exists for cgroup reasons, ignore it here.
		return nil
	}
	dest := filepath.Join(rootfs, node.Path)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		log.Errorf("createDeviceNode MkdirAll %s error %v", filepath.Dir(dest), err)
		return err
	}
	if bind {
		return bindMountDeviceNode(dest, node)
	}
	if err := mknodDevice(dest, node); err != nil {
		if os.IsExist(err) {
			return nil
		} else if os.IsPermission(err) {
			return bindMountDeviceNode(dest, node)
		}
		log.Errorf("createDeviceNode mknodDevice %s %s error %v", dest, node, err)
		return err
	}
	return nil
}

func bindMountDeviceNode(dest string, node *Device) error {
	f, err := os.Create(dest)
	if err != nil && !os.IsExist(err) {
		log.Errorf("bindMountDeviceNode %s %s error %v", dest, node, err)
		return err
	}
	if f != nil {
		f.Close()
	}
	err = unix.Mount(node.Path, dest, "bind", unix.MS_BIND, "")
	if err != nil {
		log.Errorf("bindMountDeviceNode unix.Mount %s %s error %v", node.Path, dest, err)
	}
	return err
}

//umount设备
func DelDefaultDevice(id string) {
	//mergedDirPath := getMergedPath(id)
	//if _, err := exec.Command("umount", path.Join(mergedDirPath, "/dev")).CombinedOutput(); err != nil {
	//	log.Errorf("umount  /dev failed. %v", err)
	//}
}

func mknodDevice(dest string, node *Device) error {
	fileMode := node.FileMode
	switch node.Type {
	case BlockDevice:
		fileMode |= unix.S_IFBLK
	case CharDevice:
		fileMode |= unix.S_IFCHR
	case FifoDevice:
		fileMode |= unix.S_IFIFO
	default:
		return fmt.Errorf("%c is not a valid device type for device %s", node.Type, node.Path)
	}
	dev, err := node.Mkdev()
	if err != nil {
		return err
	}
	if err := unix.Mknod(dest, uint32(fileMode), int(dev)); err != nil {
		return err
	}
	return unix.Chown(dest, int(node.Uid), int(node.Gid))
}

//https://github.com/opencontainers/runc/blob/master/libcontainer/specconv/spec_linux.go#L65
var AllowedDevices = []*Device{
	// allow mknod for any device
	{
		Path:     "/dev/null",
		FileMode: 0666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       3,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/random",
		FileMode: 0666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       8,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/full",
		FileMode: 0666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       7,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/tty",
		FileMode: 0666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       5,
			Minor:       0,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/zero",
		FileMode: 0666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       5,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	{
		Path:     "/dev/urandom",
		FileMode: 0666,
		Uid:      0,
		Gid:      0,
		Rule: Rule{
			Type:        CharDevice,
			Major:       1,
			Minor:       9,
			Permissions: "rwm",
			Allow:       true,
		},
	},
	// /dev/pts/ - pts namespaces are "coming soon"
	//{
	//	Rule: Rule{
	//		Type:        CharDevice,
	//		Major:       136,
	//		Minor:       Wildcard,
	//		Permissions: "rwm",
	//		Allow:       true,
	//	},
	//},
	//{
	//	Rule: Rule{
	//		Type:        CharDevice,
	//		Major:       5,
	//		Minor:       2,
	//		Permissions: "rwm",
	//		Allow:       true,
	//	},
	//},
	//// tuntap
	//{
	//	Rule: Rule{
	//		Type:        CharDevice,
	//		Major:       10,
	//		Minor:       200,
	//		Permissions: "rwm",
	//		Allow:       true,
	//	},
	//},
}
