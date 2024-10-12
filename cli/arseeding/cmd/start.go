/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/everFinance/arseeding"
	"github.com/everFinance/goar/types"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const pidFile string = ".arseeding_pid.lock"

var daemon bool

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start arseeding",
	Long:  `start arseeding`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if daemon {
			if _, err := os.Stat(pidFile); err == nil {
				fmt.Println("Failed start, PID file exist.running...")
				return nil
			}

			path, err := os.Executable()
			if err != nil {
				return err
			}

			command := exec.Command(path, "start")

			// add log
			logFileName := fmt.Sprintf("arseeding_%d.log", time.Now().Unix())
			logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				return err
			}

			command.Stdout = logFile
			command.Stderr = logFile

			if err := command.Start(); err != nil {
				return err
			}
			err = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", command.Process.Pid)), 0666)
			if err != nil {
				return err
			}

			daemon = false
			os.Exit(0)
		} else {
			runServer()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().BoolVarP(&daemon, "deamon", "d", false, "is daemon?")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runServer() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, os.Kill)

	useSqlite := false
	sqliteDir := ""

	tagJs := cfg.Tags
	if len(tagJs) == 0 {
		tagJs = `{"Community":"PermaDAO","Website":"permadao.com"}`
	}
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

	m := arseeding.New(cfg.BoltDir, cfg.Mysql, sqliteDir, useSqlite,
		cfg.RollupKeyPath, cfg.ArNode, cfg.CuUrl, cfg.Pay, cfg.NoFee, cfg.Manifest,
		cfg.S3KV.UseS3, cfg.S3KV.AccKey, cfg.S3KV.SecretKey, cfg.S3KV.Prefix, cfg.S3KV.Region, cfg.S3KV.Endpoint, cfg.S3KV.User4Ever,
		cfg.AliyunKV.UseAliyun, cfg.AliyunKV.Endpoint, cfg.AliyunKV.AccKey, cfg.AliyunKV.SecretKey, cfg.AliyunKV.Prefix,
		cfg.MongoDBKV.UseMongoDB, cfg.MongoDBKV.Uri,
		cfg.Port, customTags,
		cfg.Kafka.Start, cfg.Kafka.Uri)

	m.Run(cfg.Port, cfg.BundleInterval)

	<-signals
	m.Close()
}
