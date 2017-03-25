package main

import (
	"net/url"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Gocursive"
	app.Usage = "Recursive autoindex downloader"
	app.Version = VERSION

	app.Commands = []cli.Command{
		{
			Name:    "get",
			Aliases: []string{"g"},
			Usage:   "Start",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "debug, d"},
				cli.IntFlag{Name: "concurrent, c", Value: 20},
				cli.StringFlag{Name: "output-dir, o", Value: "."},
				cli.IntFlag{Name: "cpus", Value: runtime.NumCPU(), Usage: "Number of CPUs to use"},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() == 0 {
					return cli.NewExitError("URL must be given", 1)
				}

				requrl := c.Args().Get(0)
				conns := c.Int("concurrent")
				target := c.String("output-dir")
				cpus := c.Int("cpus")

				_, err := url.ParseRequestURI(requrl)
				if err != nil {
					return cli.NewExitError("Invalid URL format", 1)
				}

				if conns < 1 {
					return cli.NewExitError("The value concurrent should be greater than 0", 1)
				}

				runtime.GOMAXPROCS(cpus)

				log.Info("URL: ", requrl)
				log.Info("Connections: ", conns)
				log.Info("Target directory: ", target)
				log.Info("Number of CPUs to use: ", cpus)

				if c.Bool("debug") {
					logrus.SetLevel(logrus.DebugLevel)
					log.Info("Enabled debug mode")
				}

				config := &ClientConfig{
					url:        requrl,
					concurrent: conns,
					outputdir:  target,
				}

				client := NewClient(config)
				client.Run()
				return nil
			},
		},
	}

	app.Run(os.Args)
}
