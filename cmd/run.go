/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fireman/internal/database"
	"fireman/internal/engine"
	"fireman/internal/util"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "执行流程",
	Long:  `用于批量登陆资源执行相关命令`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.NewDB()
		if err != nil {
			util.PrintErr(err)
		}
		e := engine.NewEngine(db)
		e.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
