package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

const usage = `自己实现Docker，代码参考 https://github.com/xianlubird/mydocker`
const usageText = `类似Docker命令`

func main() {
	app := cli.NewApp()
	app.Name = "rocker"
	app.Usage = usage
	//app.UsageText = usageText
	app.Commands = []*cli.Command{
		initCommand(),
		runCommand(),
		commitCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
