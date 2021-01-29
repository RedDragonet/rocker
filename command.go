package main

import (
	"fmt"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: `初始化容器，禁止外部调用`,
		Action: func(context *cli.Context) error {
			return container.NewInitProcess()
		},
	}
}

func runCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: `创建一个带命名空间的容器`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "i",
				Usage: "开启交互模式",
			},
			&cli.BoolFlag{
				Name:  "t",
				Usage: "虚拟控制台",
			},
			&cli.BoolFlag{
				Name:  "d",
				Usage: "后台运行",
			},
			&cli.StringFlag{
				Name:  "v",
				Usage: "挂载volume",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "指定容器名称",
			},
			&cli.StringFlag{
				Name:  "m",
				Usage: "内存上限",
			},
			&cli.StringFlag{
				Name:  "cpuset",
				Usage: "指定Cpu",
			},
			&cli.StringFlag{
				Name:  "cpushare",
				Usage: "指定Cpu占用率",
			},
		},
		Action: func(context *cli.Context) error {
			if context.Args().Len() < 1 {
				return fmt.Errorf("缺少参数")
			}
			cmd := context.Args().Get(0)
			interactive := context.Bool("i")
			tty := context.Bool("t")
			volume := context.String("v")
			detach := context.Bool("d")
			containerName := context.String("name")

			if detach && interactive {
				return fmt.Errorf("交互模式，与后台运行模式不能共存")
			}

			resConf := &subsystem.ResourceConfig{
				MemoryLimit: context.String("m"),
				CpuSet:      context.String("cpuset"),
				CpuShare:    context.String("cpushare"),
			}

			log.Infof("命令 %s，参数 %b,%b", cmd, interactive, tty)
			Run(interactive, tty, volume, context.Args().Slice(), resConf, containerName)
			return nil
		},
	}
}

func commitCommand() *cli.Command {
	return &cli.Command{
		Name:  "commit",
		Usage: `打包镜像`,
		Action: func(context *cli.Context) error {
			return commit(context.Args().Get(0), context.Args().Get(1))
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "ps",
		Usage: `列出所有的镜像`,
		Action: func(context *cli.Context) error {
			ListContainers()
			return nil
		},
	}
}

func logCommand() *cli.Command {
	return &cli.Command{
		Name:  "log",
		Usage: `显示日志`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "f",
				Usage: "持续跟踪最新的日志",
			},
		},
		Action: func(context *cli.Context) error {
			follow := context.Bool("f")
			logContainer(context.Args().Get(0), follow)
			return nil
		},
	}
}

func execCommand() *cli.Command {
	return &cli.Command{
		Name:  "exec",
		Usage: `在容器中运行命令`,
		Action: func(context *cli.Context) error {
			//This is for callback
			if os.Getenv(ENV_EXEC_PID) != "" {
				log.Infof("pid callback pid %s", os.Getgid())
				return nil
			}

			if context.Args().Len() < 2 {
				return fmt.Errorf("Missing container name or command")
			}
			containerName := context.Args().Get(0)
			var commandArray []string
			for _, arg := range context.Args().Tail() {
				commandArray = append(commandArray, arg)
			}
			ExecContainer(containerName, commandArray)
			return nil
		},
	}
}
