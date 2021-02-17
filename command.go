package main

import (
	"fmt"
	"github.com/RedDragonet/rocker/cgroup/subsystem"
	"github.com/RedDragonet/rocker/container"
	"github.com/RedDragonet/rocker/image"
	"github.com/RedDragonet/rocker/network"
	log "github.com/RedDragonet/rocker/pkg/pidlog"
	"github.com/urfave/cli/v2"
	"os"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: `初始化容器，禁止外部调用`,
		Action: func(context *cli.Context) error {
			return container.NewInitProcess(context.Args().Slice())
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
			&cli.StringSliceFlag{
				Name:  "v",
				Usage: "挂载volume",
			},
			&cli.StringSliceFlag{
				Name:  "p",
				Usage: "端口映射",
			},
			&cli.StringFlag{
				Name:  "net",
				Usage: "网卡",
			},
			&cli.StringSliceFlag{
				Name:  "e",
				Usage: "环境变量",
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
			volumes := context.StringSlice("v")
			portMapping := context.StringSlice("p")
			detach := context.Bool("d")
			environ := context.StringSlice("e")
			containerName := context.String("name")
			net := context.String("net")

			if detach && interactive {
				return fmt.Errorf("交互模式，与后台运行模式不能共存")
			}

			resConf := &subsystem.ResourceConfig{
				MemoryLimit: context.String("m"),
				CpuSet:      context.String("cpuset"),
				CpuShare:    context.String("cpushare"),
			}

			log.Infof("命令 %s，参数 interactive=%v, tty=%v", cmd, interactive, tty)
			Run(interactive, tty, net, volumes, portMapping, environ, context.Args().Slice(), resConf, containerName)
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
		Usage: `列出所有的容器`,
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
				return fmt.Errorf("Missing container name or command %v", context.Args().Slice())
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

func stopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: `停止容器`,
		Action: func(context *cli.Context) error {
			StopContainer(context.Args().Get(0))
			return nil
		},
	}
}

func removeCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: `删除容器`,
		Action: func(context *cli.Context) error {
			RemoveContainer(context.Args().Get(0))
			return nil
		},
	}
}

func networkCommand() *cli.Command {
	return &cli.Command{
		Name:  "network",
		Usage: "网络",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "创建网卡",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "driver",
						Usage: "driver",
					},
					&cli.StringFlag{
						Name:  "subnet",
						Usage: "子网IP",
					},
				},
				Action: func(context *cli.Context) error {
					if context.Args().Len() < 1 {
						return fmt.Errorf("参数缺失")
					}
					if err := network.Init(); err != nil {
						return err
					}
					//创建网络设备
					err := network.CreateNetwork(context.String("driver"), context.String("subnet"), context.Args().Get(0))
					if err != nil {
						return fmt.Errorf("创建网络失败: %+v", err)
					}
					return nil
				},
			},
			{
				Name:  "list",
				Usage: "列出所有已经创建的网络设备",
				Action: func(context *cli.Context) error {
					if err := network.Init(); err != nil {
						return err
					}

					network.ListNetwork()
					return nil
				},
			},
			{
				Name:  "remove",
				Usage: "移除网络设备",
				Action: func(context *cli.Context) error {
					if context.Args().Len() < 1 {
						return fmt.Errorf("参数缺失")
					}
					if err := network.Init(); err != nil {
						return err
					}
					err := network.DeleteNetwork(context.Args().Get(0))
					if err != nil {
						return fmt.Errorf("删除 network 失败: %+v", err)
					}
					return nil
				},
			},
		},
	}
}

func pullCommand() *cli.Command {
	return &cli.Command{
		Name:  "pull",
		Usage: `镜像拉取`,
		Action: func(context *cli.Context) error {
			if context.Args().Len() < 1 {
				return fmt.Errorf("缺少镜像名称")
			}
			err := image.Pull(context.Args().Get(0))
			if err != nil {
				log.Errorf("镜像拉取失败 %v", err)
			}
			return nil
		},
	}
}

func imagesCommand() *cli.Command {
	return &cli.Command{
		Name:  "images",
		Usage: "镜像列表",
		Action: func(context *cli.Context) error {
			image.Init()

			image.ListImage()
			return nil
		},
	}
}
func imageCommand() *cli.Command {
	return &cli.Command{
		Name:  "image",
		Usage: "镜像",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "列出所有镜像",
				Action: func(context *cli.Context) error {
					image.Init()

					image.ListImage()
					return nil
				},
			},
			{
				Name:  "remove",
				Usage: "移除镜像",
				Action: func(context *cli.Context) error {
					if context.Args().Len() < 1 {
						return fmt.Errorf("参数缺失")
					}
					image.Init()
					err := image.Delete(context.Args().Get(0))
					if err != nil {
						return fmt.Errorf("删除 镜像 失败: %+v", err)
					}
					return nil
				},
			},
		},
	}
}
