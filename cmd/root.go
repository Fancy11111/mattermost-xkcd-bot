/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/Fancy11111/mattermost-xkcd-bot/bot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "mattermost-xkcd-bot",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		Run: func(cmd *cobra.Command, args []string) {
			bot, err := bot.NewBot()
			if err != nil {
				os.Exit(1)
			}
			bot.Start()
		},
	}
)

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

	cobra.OnInitialize(initConfig)
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mattermost-xkcd-bot.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.PersistentFlags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file")
	rootCmd.PersistentFlags().StringP("serverUrl", "u", "", "ServerUrl")
	rootCmd.PersistentFlags().StringP("botId", "i", "", "Bot ID")
	rootCmd.PersistentFlags().StringP("botToken", "t", "", "Bot Token")
	rootCmd.PersistentFlags().StringP("logChannel", "l", "", "Log channel")
	rootCmd.PersistentFlags().StringP("team", "T", "", "Bot Team")
	rootCmd.PersistentFlags().StringP("prefix", "p", "!xkcd", "Bot Team")

	BindPFlag(rootCmd, "serverUrl")
	BindPFlag(rootCmd, "botId")
	BindPFlag(rootCmd, "botToken")
	BindPFlag(rootCmd, "logChannel")
	BindPFlag(rootCmd, "team")
	BindPFlag(rootCmd, "prefix")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	}
	// else {
	// 	cwd, err := os.Getwd()
	// 	cobra.CheckErr(err)
	//
	// 	// Search config in home directory with name ".cobra" (without extension).
	// 	viper.AddConfigPath(cwd)
	// 	viper.SetConfigType("yaml")
	// 	viper.SetConfigName(".bot")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func BindPFlag(cmd *cobra.Command, name string) {
	viper.BindPFlag(name, rootCmd.PersistentFlags().Lookup(name))
}
