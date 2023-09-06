/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "fireman",
	Short:   "主机应急响应工具",
	Version: version,
	Long: ` 
███████╗██╗██╗     ███████╗███╗   ███╗ █████╗ ███╗   ██╗
██╔════╝██║██║     ██╔════╝████╗ ████║██╔══██╗████╗  ██║
█████╗  ██║██║     █████╗  ██╔████╔██║███████║██╔██╗ ██║
██╔══╝  ██║██║     ██╔══╝  ██║╚██╔╝██║██╔══██║██║╚██╗██║
██║     ██║███████╗███████╗██║ ╚═╝ ██║██║  ██║██║ ╚████║
╚═╝     ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
		作者：xiaoliu	版本：1.0
		用于批量排查服务器状态信息的工具
`,
	//Uncomment the following line if your bare application
	//has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.4ATOOLS.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.PersistentFlags().StringVarP(&config.Cookie, "Cookie", "c", "", "authority user cookie, it is sid")
}
