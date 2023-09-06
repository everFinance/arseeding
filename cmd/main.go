package main

import (
	"encoding/json"
	"github.com/everFinance/arseeding"
	"github.com/everFinance/arseeding/common"
	"github.com/everFinance/goar/types"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mkevac/debugcharts"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "Arseeding",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "db_dir", Value: "./data/bolt", Usage: "bolt db dir path", EnvVars: []string{"DB_DIR"}},
			&cli.StringFlag{Name: "mysql", Value: "root@tcp(127.0.0.1:3306)/arseed?charset=utf8mb4&parseTime=True&loc=Local", Usage: "mysql dsn", EnvVars: []string{"MYSQL"}},
			&cli.BoolFlag{Name: "use_sqlite", Value: false, EnvVars: []string{"USE_SQLITE"}},
			&cli.StringFlag{Name: "sqlite_dir", Value: "./data/sqlite", Usage: "sqlite db dir path", EnvVars: []string{"SQLITE_DIR"}},

			&cli.StringFlag{Name: "key_path", Value: "./data/bundler-keyfile.json", Usage: "ar keyfile path", EnvVars: []string{"KEY_PATH"}},
			&cli.StringFlag{Name: "ar_node", Value: "https://arweave.net", EnvVars: []string{"AR_NODE"}},
			&cli.StringFlag{Name: "pay", Value: "https://api-dev.everpay.io", Usage: "pay url", EnvVars: []string{"PAY"}},
			&cli.BoolFlag{Name: "no_fee", Value: false, EnvVars: []string{"NO_FEE"}},
			&cli.BoolFlag{Name: "manifest", Value: true, EnvVars: []string{"MANIFEST"}},
			&cli.IntFlag{Name: "bundle_interval", Value: 120, Usage: "bundle tx on chain time interval(seconds)", EnvVars: []string{"BUNDLE_INTERVAL"}},

			&cli.BoolFlag{Name: "use_s3", Value: false, Usage: "run with s3 store", EnvVars: []string{"USE_S3"}},
			&cli.StringFlag{Name: "s3_acc_key", Value: "AKIATZSGGOHIV4QTYNH5", Usage: "s3 access key", EnvVars: []string{"S3_ACC_KEY"}},
			&cli.StringFlag{Name: "s3_secret_key", Value: "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", Usage: "s3 secret key", EnvVars: []string{"S3_SECRET_KEY"}},
			&cli.StringFlag{Name: "s3_prefix", Value: "arseed", Usage: "s3 bucket name prefix", EnvVars: []string{"S3_PREFIX"}},
			&cli.StringFlag{Name: "s3_region", Value: "ap-northeast-1", Usage: "s3 bucket region", EnvVars: []string{"S3_REGION"}},
			&cli.StringFlag{Name: "s3_endpoint", Value: "", Usage: "s3 bucket endpoint", EnvVars: []string{"S3_ENDPOINT"}},
			&cli.BoolFlag{Name: "use_4ever", Value: false, Usage: "run with 4everland s3 service", EnvVars: []string{"USE_4EVER"}},

			&cli.BoolFlag{Name: "use_aliyun", Value: false, Usage: "run with aliyun oss store", EnvVars: []string{"USE_ALIYUN"}},
			&cli.StringFlag{Name: "aliyun_endpoint", Value: "oss-cn-shanghai.aliyuncs.com", Usage: "aliyun oss endpoint", EnvVars: []string{"ALIYUN_ENDPOINT"}},
			&cli.StringFlag{Name: "aliyun_acc_key", Value: "your oss access key", Usage: "aliyun oss access key", EnvVars: []string{"ALIYUN_ACC_KEY"}},
			&cli.StringFlag{Name: "aliyun_secret_key", Value: "your oss secret key", Usage: "aliyun oss secret key", EnvVars: []string{"ALIYUN_SECRET_KEY"}},
			&cli.StringFlag{Name: "aliyun_prefix", Value: "arseed", Usage: "aliyun oss bucket name prefix", EnvVars: []string{"ALIYUN_PREFIX"}},

			&cli.BoolFlag{Name: "use_mongodb", Value: false, Usage: "run with mongo db store", EnvVars: []string{"USE_MONGODB"}},
			&cli.StringFlag{Name: "mongodb_uri", Value: "mongodb://user:password@localhost:27017", Usage: "mongo uri", EnvVars: []string{"MONGODB_URI"}},

			&cli.StringFlag{Name: "port", Value: ":8080", EnvVars: []string{"PORT"}},
			&cli.StringFlag{Name: "tags", Value: `{"Community":"PermaDAO","Website":"permadao.com"}`, EnvVars: []string{"TAGS"}},

			// kafka
			&cli.BoolFlag{Name: "use_kafka", Value: false, Usage: "kafka used", EnvVars: []string{"USE_KAFKA"}},
			&cli.StringFlag{Name: "kafka_uri", Value: "34.220.174.25:9092", Usage: "kafka uri", EnvVars: []string{"KAFKA_URI"}},
			// &cli.StringFlag{Name: "kafka_uri", Value: "kafka.corp.knn3.xyz:19092", Usage: "kafka uri", EnvVars: []string{"KAFKA_URI"}},
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

	tagJs := c.String("tags")
	tagsMap := make(map[string]string)
	if err := json.Unmarshal([]byte(tagJs), &tagsMap); err != nil {
		panic(err)
	}
	customTags := make([]types.Tag, 0)
	for k, v := range tagsMap {
		customTags = append(customTags, types.Tag{
			Name:  k,
			Value: v,
		})
	}

	s := arseeding.New(
		c.String("db_dir"), c.String("mysql"), c.String("sqlite_dir"), c.Bool("use_sqlite"),
		c.String("key_path"), c.String("ar_node"), c.String("pay"), c.Bool("no_fee"), c.Bool("manifest"),
		c.Bool("use_s3"), c.String("s3_acc_key"), c.String("s3_secret_key"), c.String("s3_prefix"), c.String("s3_region"), c.String("s3_endpoint"),
		c.Bool("use_4ever"), c.Bool("use_aliyun"), c.String("aliyun_endpoint"), c.String("aliyun_acc_key"), c.String("aliyun_secret_key"), c.String("aliyun_prefix"),
		c.Bool("use_mongodb"), c.String("mongodb_uri"),
		c.String("port"), customTags,
		c.Bool("use_kafka"), c.String("kafka_uri"))
	s.Run(c.String("port"), c.Int("bundle_interval"))

	common.NewMetricServer()

	<-signals

	s.Close()
	return nil
}
