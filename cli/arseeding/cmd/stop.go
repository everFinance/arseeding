/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop arseeding",
	Long:  `stop arseeding`,
	RunE: func(cmd *cobra.Command, args []string) error {
		strb, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("Stop server failed, err: %v", err)
			return nil
		}
		command := exec.Command("kill", string(strb))
		if err := command.Start(); err != nil {
			return err
		}
		if err := os.Remove(pidFile); err != nil {
			return err
		}
		println("arseeding stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stopCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stopCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
