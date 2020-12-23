package container

import (
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

//容器初始化命令
func NewInitProcess(command string, args []string) error {

	log.Infof("命令 %s，参数 %+q", command, args)
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	//挂载 Proc 目录
	log.Infof("挂载 Proc 目录")
	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		log.Errorf("挂载 Proc 失败", err)
		return err
	}

	//挂载 devpts
	//err = syscall.Mount("devpts", "/dev/pts", "devpts", uintptr(defaultMountFlags), "")
	//if err != nil {
	//	log.Errorf("挂载 devpts 失败", err)
	//	return err
	//}

	logrus.Infof("syscall.Exec 开始，command=%s args=%v", command, args)
	if err := syscall.Exec(command, args, os.Environ()); err != nil {
		logrus.Errorf("syscall.Exec Error %s", err.Error())
		return err
	}
	return nil
}

func NewParentProcess(interactive, tty bool, command string, args []string) *exec.Cmd {
	//首先调用自己的初始化命令
	args = append([]string{"init", command}, args...)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}

	//交互模式
	if interactive {
		cmd.Stdin = os.Stdin
	}

	//虚拟控制台
	if tty {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}
