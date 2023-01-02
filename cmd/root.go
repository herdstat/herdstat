/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package cmd

import (
	"fmt"
	"go.uber.org/zap"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// The configuration file used to initialize flags and parameters
	cfgFile string

	// Flag to enable verbose output
	verbose bool
)
var logger *zap.SugaredLogger

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "herdstat",
	Short: "stat tool for open source communities",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger = configureLogger()
	},
}

// configureLogger configures the logging subsystem
func configureLogger() *zap.SugaredLogger {
	var config zap.Config
	if verbose {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	l, err := config.Build()
	logger = l.Sugar()
	if err != nil {
		logger.Panicw("Log system initialization failed", "Error", err)
	}
	if verbose {
		logger.Infow("Verbose output enabled")
	}
	return logger
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// Initialize the root command.
func init() {
	cobra.OnInitialize(initConfig)

	// Flag to specify config file to be used.
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default is $HOME/.herdstat.yaml)")

	// Flag to enable verbose output
	rootCmd.PersistentFlags().BoolVar(
		&verbose,
		"verbose",
		false,
		"enable verbose output")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".herdstat")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
