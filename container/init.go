package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

//容器初始化命令
func NewInitProcess() error {
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
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		log.Errorf("读取管道参数异常 %v", err)
		return fmt.Errorf("读取管道参数异常 %v", err)
	}
	command := cmdArray[0]
	log.Infof("命令 %s", command)

	command, err = exec.LookPath(command)
	if err != nil {
		log.Errorf("命令 %s 查找失败 %v ", command, err)
		return err
	}
	log.Infof("命令查找成功 %s", command)

	logrus.Infof("syscall.Exec 开始，command=%s", command)
	if err := syscall.Exec(command, cmdArray[0:], os.Environ()); err != nil {
		logrus.Errorf("syscall.Exec Error %s", err.Error())
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

func NewParentProcess(interactive, tty bool) (*exec.Cmd, *os.File) {
	//首先调用自己的初始化命令
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
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
	}
	return cmd, write
}
