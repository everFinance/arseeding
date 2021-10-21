package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/everFinance/seeding"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "seeding",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "port", Value: ":8080", EnvVars: []string{"PORT"}},
		},
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	s := seeding.New()
	s.Run(c.String("port"))

	<-signals

	return nil
}