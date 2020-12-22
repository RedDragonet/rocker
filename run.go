package main

import (
	"github.com/RedDragonet/rocker/container"
	log "github.com/sirupsen/logrus"
	"os"
)

func Run(interactive, tty bool, cmd string) {
	parent := container.NewParentProcess(interactive, tty, cmd)
	if parent == nil {
		log.Errorf("创建父进程失败")
		return
	}

	log.Infof("当前进程ID", os.Getpid())

	if err := parent.Start(); err != nil {
		log.Infof("父进程运行失败")
	}
	log.Infof("创建父运行成功，开始等待")
	log.Infof("当前进程ID", os.Getpid())
	_ = parent.Wait()
	log.Infof("父进程运行结束")
	os.Exit(0)
}
