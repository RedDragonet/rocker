package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

const usage = `自己实现Docker，代码参考 https://github.com/xianlubird/mydocker`

func main() {
	app := cli.NewApp()
	app.Name = "rocker"
	app.Usage = usage
	app.Commands = []*cli.Command{
		initCommand(),
		runCommand(),
		commitCommand(),
		listCommand(),
		logCommand(),
		execCommand(),
		stopCommand(),
		removeCommand(),
		networkCommand(),
		pullCommand(),
		imageCommand(),
		imagesCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
