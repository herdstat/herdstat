/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v48/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/url"
	"os"
	"regexp"
)

// Configuration keys for the root command
const (
	// Repositories to analyze
	repositoriesCfgKey = "repositories"
	// Toggle for verbose output
	verboseCfgKey = "verbose"
)

var (
	// The file to read the configuration from
	cfgFile string
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
	verbose := viper.GetBool(verboseCfgKey)
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

// Matches GitHub owner or repository identifiers (see
// https://github.com/dead-claudia/github-limits for details)
var ownerOrRepoIdPattern = regexp.MustCompile(fmt.Sprintf("([A-Za-z0-9-]+)(/([A-Za-z0-9_\\.-]+))?"))

// addRepository adds the URL built from the given owner and repository to the
// given set of URLs.
func addRepository(owner string, repo string, urls *map[url.URL]bool) error {
	repoUrl, err := url.Parse(fmt.Sprintf("https://github.com/%s/%s", owner, repo))
	if err != nil {
		return err
	}
	if _, ok := (*urls)[*repoUrl]; ok {
		logger.Warnw("Repository is a duplicate - ignoring", "Repository URL", repoUrl.String())
	} else {
		(*urls)[*repoUrl] = true
	}
	return nil
}

// addOwnerRepositories fetches all repositories of the given owner and adds
// their URL to the given set of URLs.
func addOwnerRepositories(owner string, urls *map[url.URL]bool) error {
	client := github.NewClient(nil)
	opt := &github.RepositoryListByOrgOptions{Type: "public"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), owner, opt)
	logger.Infow("Fetched repositories from owner", "Owner", owner, "Count", len(repos))
	if err != nil {
		return err
	}
	for _, repo := range repos {
		if err := addRepository(owner, *repo.Name, urls); err != nil {
			return err
		}
	}
	return nil
}

// resolveRepositoryURLs computes the repositories to be analyzed. Performs
// expansion of owner entries and deduplication.
func resolveRepositoryURLs() (map[url.URL]bool, error) {
	repos := viper.GetStringSlice(repositoriesCfgKey)
	urls := make(map[url.URL]bool)
	for _, repo := range repos {
		matches := ownerOrRepoIdPattern.FindStringSubmatch(repo)
		if matches == nil {
			return nil, fmt.Errorf("'%s' is not a valid owner or owner/repository", repo)
		}
		owner := matches[1]
		if matches[3] == "" {
			err := addOwnerRepositories(owner, &urls)
			if err != nil {
				return nil, fmt.Errorf("failed to collect repositories from owner '%s': %w", owner, err)
			}
		} else {
			repository := matches[3]
			err := addRepository(owner, repository, &urls)
			if err != nil {
				return nil, fmt.Errorf("failed to add repository '%s': %w", repository, err)
			}
		}
	}
	if len(urls) == 0 {
		return nil, errors.New("resolving repositories resulted in empty set")
	}
	return urls, nil
}

// Initialize the root command.
func init() {
	cobra.OnInitialize(initConfig)

	// Flag to specify config file to be used.
	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config",
		"c",
		"",
		"config file (default is $HOME/.herdstat.yaml)")

	// Flag to enable verbose output
	const verboseFlag = "verbose"
	rootCmd.PersistentFlags().BoolP(
		verboseFlag,
		"v",
		false,
		"enable verbose output")
	if err := viper.BindPFlag(verboseCfgKey, rootCmd.PersistentFlags().Lookup(verboseFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", verboseFlag, "Error", err)
	}

	// Flag to specify repositories to analyze
	const repositoriesFlag = "repositories"
	rootCmd.PersistentFlags().StringSliceP(
		repositoriesFlag,
		"r",
		nil,
		"repositories to analyze",
	)
	if err := viper.BindPFlag(repositoriesCfgKey, rootCmd.PersistentFlags().Lookup(repositoriesFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", repositoriesFlag, "Error", err)
	}

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
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
