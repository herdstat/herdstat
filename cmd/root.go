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
	"github.com/google/go-github/v50/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Configuration keys for the root command
const (

	// Repositories to analyze
	repositoriesCfgKey = "repositories"

	// Token used to access the GitHub API
	gitHubTokenCfgKey = "github-token"

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
var ownerOrRepoIDPattern = regexp.MustCompile(fmt.Sprintf("([A-Za-z0-9-]+)(/([A-Za-z0-9_\\.-]+))?"))

// getHTTPClient returns a http client that uses a GitHub token for authentication
// if configured through viper.
func getHTTPClient() *http.Client {
	var httpClient *http.Client
	if viper.IsSet(gitHubTokenCfgKey) {
		token := viper.GetString(gitHubTokenCfgKey)
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		ctx := context.Background()
		httpClient = oauth2.NewClient(ctx, ts)
		logger.Debug("GitHub token provided - making authenticated API calls")
	} else {
		httpClient = http.DefaultClient
		logger.Debug("No GitHub token provided - making anonymous API calls")
	}
	return httpClient
}

// addRepository adds the repository given by repository owner and name to the map of repositories.
func addRepositoryFromName(owner string, repo string, repositories *map[url.URL]*github.Repository) error {
	client := github.NewClient(getHTTPClient())
	repository, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		return err
	}
	return addRepository(repository, repositories)
}

// addRepository adds the given repository to the given map of repositories, if it is not a duplicate.
func addRepository(repo *github.Repository, repositories *map[url.URL]*github.Repository) error {
	repoURL, err := url.Parse(repo.GetHTMLURL())
	if err != nil {
		return err
	}
	if _, ok := (*repositories)[*repoURL]; ok {
		logger.Warnw("Repository is a duplicate - ignoring", "Repository URL", repoURL.String())
	} else {
		(*repositories)[*repoURL] = repo
	}
	return nil
}

// addOwnedRepositories fetches all repositories of the given owner and adds
// them to the given map.
func addOwnedRepositories(owner string, repositories *map[url.URL]*github.Repository) error {
	client := github.NewClient(getHTTPClient())
	opt := &github.RepositoryListByOrgOptions{Type: "public"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), owner, opt)
	logger.Debugw("Fetched repositories from owner", "Owner", owner, "Count", len(repos))
	if err != nil {
		return err
	}
	for _, repo := range repos {
		if err := addRepository(repo, repositories); err != nil {
			return err
		}
	}
	return nil
}

// collectRepositories computes the repositories to be analyzed. Performs
// expansion of owner entries and deduplication.
func collectRepositories() (map[url.URL]*github.Repository, error) {
	repos := viper.GetStringSlice(repositoriesCfgKey)
	repositories := make(map[url.URL]*github.Repository)
	for _, repo := range repos {
		matches := ownerOrRepoIDPattern.FindStringSubmatch(repo)
		if matches == nil {
			return nil, fmt.Errorf("'%s' is not a valid owner or owner/repository", repo)
		}
		owner := matches[1]
		if matches[3] == "" {
			err := addOwnedRepositories(owner, &repositories)
			if err != nil {
				return nil, fmt.Errorf("failed to collect repositories from owner '%s': %w", owner, err)
			}
		} else {
			repository := matches[3]
			err := addRepositoryFromName(owner, repository, &repositories)
			if err != nil {
				return nil, fmt.Errorf("failed to add repository '%s': %w", repository, err)
			}
		}
	}
	if len(repositories) == 0 {
		return nil, errors.New("resolving repositories resulted in empty set")
	}
	return repositories, nil
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

	// Flag to set the access token used for making GitHub API calls
	const gitHubTokenFlag = "github-token"
	rootCmd.PersistentFlags().StringP(
		gitHubTokenFlag,
		"t",
		"",
		"token for accessing the GitHub API",
	)
	if err := viper.BindPFlag(gitHubTokenCfgKey, rootCmd.PersistentFlags().Lookup(gitHubTokenFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", gitHubTokenFlag, "Error", err)
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
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
