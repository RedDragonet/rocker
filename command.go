package main

import (
	"fmt"
	"github.com/RedDragonet/rocker/container"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: `初始化容器，禁止外部调用`,
		Action: func(context *cli.Context) error {
			log.Infof("初始化开始")

			cmd := context.Args().Get(0)
			log.Infof("初始化命令 %s", cmd)

			return container.NewInitProcess(cmd, nil)
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
		},
		Action: func(context *cli.Context) error {
			if context.Args().Len() < 1 {
				return fmt.Errorf("缺少参数")
			}
			cmd := context.Args().Get(0)
			interactive := context.Bool("i")
			tty := context.Bool("t")

			log.Infof("命令 %s,参数 %b,%b", cmd, interactive, tty)
			Run(interactive, tty, cmd)
			return nil
		},
	}
}

