package container

import (
	"fmt"
	image2 "github.com/RedDragonet/rocker/image"
	"github.com/syndtr/gocapability/capability"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	_ "github.com/RedDragonet/rocker/nsenter"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
)

var (
	DefaultInfoLocation string = "/var/lib/rocker/containers"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
)

//容器初始化命令
func NewInitProcess(defaultArgv []string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Infof("初始化容器")
	argv := readUserCommand()
	if argv == nil || len(argv) == 0 || argv[0] == "" {
		//存在默认参数
		if len(defaultArgv) > 0 {
			//覆盖
			argv = defaultArgv
		} else {
			log.Errorf("读取管道参数异常")
			return fmt.Errorf("读取管道参数异常")
		}
	}

	command := argv[0]
	log.Infof("命令 %s %v", command, argv)

	//Init 挂载点
	err := setUpMount()
	if err != nil {
		return err
	}

	//设置默认权限
	err = applyCaps()
	if err != nil {
		log.Errorf("权限设置异常", err)
		return err
	}

	command, err = exec.LookPath(command)
	if err != nil {
		log.Errorf("命令 %s 查找失败 %v ", command, err)
		return err
	}
	log.Infof("命令查找成功 %s", command)

	log.Infof("syscall.Exec 开始，command=%s", command)

	if err := syscall.Exec(command, argv[0:], os.Environ()); err != nil {
		log.Errorf("syscall.Exec Error %s", err.Error())
		return err
	}
	return nil
}

//设置默认权限
func applyCaps() error {
	pid, err := capability.NewPid2(0)
	if err != nil {
		return err
	}

	allCapabilityTypes := capability.CAPS | capability.BOUNDS | capability.AMBS
	defaultCap, err := DefaultCapabilities()
	if err != nil {
		return err
	}

	pid.Clear(allCapabilityTypes)
	pid.Set(capability.BOUNDS, defaultCap...)
	pid.Set(capability.PERMITTED, defaultCap...)
	pid.Set(capability.INHERITABLE, defaultCap...)
	pid.Set(capability.EFFECTIVE, defaultCap...)

	//k8s 不支持 CAP_AMBIENT
	pid.Clear(capability.AMBIENT)

	return pid.Apply(allCapabilityTypes)
}

/**
Init 挂载点
*/
func setUpMount() error {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("获取当前工作目录失败 %v", err)
		return err
	}
	log.Infof("当前工作目录是 %s", pwd)

	//需要在 pivotRoot 前挂载 proc，具体的说是 pivotRoot中 unmount putold 之前
	//由于挂载 Proc 需要 ROOT 权限
	//由于设置了 CLONE_NEWUSER，运行用户无 ROOT 权限
	//需要将 CLONE_NEWPID 隔离的进程信息挂载到 newrootfs 中
	//mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	//挂载 Proc 目录
	log.Infof("挂载 Proc 目录")
	err = syscall.Mount("proc", path.Join(pwd, "/proc"), "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		return err
	}

	//err = syscall.Mount("tmpfs", path.Join(pwd, "/dev"), "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	//if err != nil {
	//	return err
	//}

	//挂载 devpts
	//err = syscall.Mount("devpts", path.Join(pwd, "/dev/pts"), "devpts", uintptr(defaultMountFlags), "")
	//if err != nil {
	//	log.Errorf("挂载 devpts 失败", err)
	//	return err
	//}

	//创建默认设备
	err = createDefaultDevice(pwd)
	if err != nil {
		return err
	}

	//创建默认设备
	err = setupDevSymlinks(pwd)
	if err != nil {
		return err
	}

	err = pivotRoot(pwd)
	if err != nil {
		return err
	}

	return nil
}

func readUserCommand() []string {
	log.Infof("开始读取用户参数")
	//uintptr(3)就是指index 为3的文件描述符，也就是传递进来的管道的一端
	pipe := os.NewFile(uintptr(3), "pipe")
	defer pipe.Close()
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		log.Errorf("初始化 read pipe 错误 %v", err)
		return nil
	}
	msgStr := string(msg)
	log.Infof("读取用户参数 %s", msgStr)
	return strings.Split(msgStr, " ")
}

func NewParentProcess(interactive, tty bool, image string, volumeSlice, environSlice []string, containerId, containerName string) (*exec.Cmd, *os.File) {
	//首先调用自己的初始化命令
	cmd := exec.Command("/proc/self/exe", "init")

	//无容器ID 为 exec命令
	//TODO:: 重构NewParentProcess函数入参
	if containerId != "" && containerName != "" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER,
			UidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getuid(),
					//TODO:: 新增用户id从1001开始，设置为1100允许容器添加99个user
					//Docker中 这个值为 unsigned int32 最大值
					//cat /proc/$$/uid_map
					//此次对于 User Namespace到理解还不够
					Size: 1100,
				},
			},
			GidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getgid(),
					Size:        1100,
				},
			},
			GidMappingsEnableSetgroups: true,
		}
	}

	//新建管道
	read, write, err := os.Pipe()
	if err != nil {
		log.Errorf("新建管道错误 %v", err)
		return nil, nil
	}

	cmd.ExtraFiles = []*os.File{
		read,
	}

	//交互模式
	if interactive {
		cmd.Stdin = os.Stdin
	}

	//虚拟控制台
	if tty {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		dirURL := path.Join(DefaultInfoLocation, containerId)
		if err := os.MkdirAll(dirURL, 0644); err != nil {
			log.Errorf("NewParentProcess mkdir %s error %v", dirURL, err)
			return nil, nil
		}
		stdLogFilePath := path.Join(dirURL, ContainerLogFile)
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			log.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
	}

	//mount overlayFS
	if image != "" {
		mntUrl, err := NewWorkSpace(image, containerId)
		if err != nil {
			log.Errorf("NewWorkSpace %s 失败 %v", image, err)
			return nil, nil
		}
		cmd.Dir = mntUrl

		i := image2.Get(image)
		r, err := i.GetRuntime()
		if err != nil {
			log.Errorf("GetRuntime %s 失败 %v", image, err)
			return nil, nil
		}

		//镜像原始配置 ENV
		environSlice = append(environSlice, r.ContainerConfig.Env...)

		//镜像默认CMD
		cmd.Args = append(cmd.Args, r.Config.Cmd...)
	}

	if len(volumeSlice) > 0 {
		err = MountVolumeSlice(cmd.Dir, volumeSlice)
		if err != nil {
			log.Errorf("mountVolumeSlice 失败 %v", err)
			return nil, nil
		}
	}

	cmd.Env = append(os.Environ(), environSlice...)

	return cmd, write
}

//建议参数 http://ifeanyi.co/posts/linux-namespaces-part-3/
func pivotRoot(rootfs string) error {
	/* Ensure that 'new_root' and its parent mount don't have
	   shared propagation (which would cause pivot_root() to
	   return an error), and prevent propagation of mount
	   events to the initial mount namespace */
	//if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
	//	log.Errorf("mount MS_PRIVATE  error ", err)
	//	os.Exit(0)
	//	return err
	//}

	//将修改 rootfs 挂载点，将rootfs 挂载到当前 mount namespace
	//满足 pivot_root 要求，old root 和 new root 需要在不同的 file system
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		log.Errorf("mount rootfs %s error ", rootfs, err)
		os.Exit(0)
		return err
	}

	//新增.putold
	putold := filepath.Join(rootfs, ".putold")
	if _, err := os.Stat(putold); !os.IsNotExist(err) {
		log.Infof("Remove putold %s", putold)
		_ = os.Remove(putold)
	}

	if err := os.Mkdir(putold, 0777); err != nil {
		return err
	}

	if err := syscall.PivotRoot(rootfs, putold); err != nil {
		log.Errorf("pivotRoot error %s => %s", rootfs, putold, err)
		return err
	}

	//修改工作目录为根节点
	if err := syscall.Chdir("/"); err != nil {
		log.Errorf("Chdir error ", err)
		return err
	}

	//将老的 rootfs(putold) 移除目录树 umount -l putold
	if err := syscall.Unmount(path.Join("/", ".putold"), syscall.MNT_DETACH); err != nil {
		log.Errorf("umount -l .putold error ", err)
		return err
	}

	//删除临时目录 .putold
	if err := os.Remove(path.Join("/", ".putold")); err != nil {
		log.Errorf("remove .putold error ", err)
		return err
	}
	return nil
}
