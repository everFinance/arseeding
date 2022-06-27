package main

import (
	"github.com/everFinance/arseeding"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "arseeding",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "db_dir", Value: "./data/bolt", Usage: "bolt db dir path", EnvVars: []string{"DB_DIR"}},
			&cli.StringFlag{Name: "mysql", Value: "root@tcp(127.0.0.1:3306)/arseed?charset=utf8mb4&parseTime=True&loc=Local", Usage: "mysql dsn", EnvVars: []string{"MYSQL"}},
			&cli.StringFlag{Name: "key_path", Value: "./data/bundler-keyfile.json", Usage: "ar keyfile path", EnvVars: []string{"KEY_PATH"}},
			&cli.StringFlag{Name: "ar_node", Value: "https://arweave.net", EnvVars: []string{"AR_NODE"}},
			&cli.StringFlag{Name: "pay", Value: "https://api-dev.everpay.io", Usage: "pay url", EnvVars: []string{"PAY"}},
			&cli.BoolFlag{Name: "s3_flag", Value: false, Usage: "s3 store", EnvVars: []string{"S3_FLAG"}},
			&cli.StringFlag{Name: "s3_accKey", Value: "AKIATZSGGOHIV4QTYNH5", Usage: "s3 store", EnvVars: []string{"S3_ACCKEY"}},
			&cli.StringFlag{Name: "s3_secretKey", Value: "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", Usage: "s3 store", EnvVars: []string{"S3_SECRETKEY"}},
			&cli.StringFlag{Name: "s3_bucketPrefix", Value: "arseed", Usage: "s3 store", EnvVars: []string{"S3_BUCKETPREFIX"}},
			&cli.StringFlag{Name: "s3_region", Value: "ap-northeast-1", Usage: "s3 store", EnvVars: []string{"S3_REGION"}},

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

	s := arseeding.New(
		c.String("db_dir"), c.String("mysql"), c.String("key_path"), c.String("ar_node"), c.String("pay"),
		c.Bool("s3_flag"), c.String("s3_accKey"), c.String("s3_secretKey"), c.String("s3_bucketPrefix"), c.String("s3_region"),
	)
	s.Run(c.String("port"))

	<-signals

	return nil
}
