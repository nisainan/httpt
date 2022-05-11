package main

import (
	"fmt"
	"github.com/nisainan/wstunnel/client"
	"github.com/nisainan/wstunnel/server"
	"github.com/urfave/cli"
	_ "go.uber.org/automaxprocs"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "wstunnel"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Value: "server",
			Usage: "run type",
		},
		cli.StringFlag{
			Name:  "config",
			Value: "$(type).yaml",
			Usage: "config file url",
		},
	}
	app.Action = func(c *cli.Context) error {
		serverType := c.String("type")
		if c.String("config") == "$(type).yaml" {
			c.Set("config", fmt.Sprintf("%s.yaml", serverType))
		}
		switch serverType {
		case "client":
			return client.Run(c)
		case "server":
			server.Run(c)
		default:
			return client.Run(c)
		}
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		panic("app run error:" + err.Error())
	}
}
