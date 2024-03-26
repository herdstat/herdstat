/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package cmd

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/araddon/dateparse"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/v50/github"
	"github.com/icza/gox/imagex/colorx"
	"github.com/repeale/fp-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/svg"
	"herdstat/internal"
	"image/color"
	"io"
	"math"
	"net/url"
	"os"
	"strings"
	"time"
)

// Configuration keys for the contribution-graph command
const (
	// Whether the output SVG should be minified
	minifyOutputCfgKey = "contribution-graph.minify"
	// The name of the output SVG file
	filenameCfgKey = "contribution-graph.filename"
	// The primary color used to color the daily contribution cells
	colorCfgKey = "contribution-graph.color"
	// The number of color levels used for coloring contribution cells
	levelsCfgKey = "contribution-graph.levels"
	// The filters used to exclude commits
	commitFiltersCfgKey = "contribution-graph.filters.commits"
	// The date of the last day to visualize
	untilCfgKey = "until"
)

// contributionGraphCmd represents the contribution-graph command
var contributionGraphCmd = &cobra.Command{
	Use:   "contribution-graph",
	Short: "Generates a GitHub-style heatmap to visualize contributions",
	Args:  cobra.NoArgs,
	RunE:  run,
}

// getUntilDate retrieves the "until" parameter as a time.Time instance by
// parsing the respective configuration entry. The date is converted to
// last nanosecond of the day.
func getUntilDate() (time.Time, error) {
	s := viper.GetString(untilCfgKey)
	date, err := dateparse.ParseStrict(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(
		date.Year(), date.Month(), date.Day(),
		23, 59, 59,
		int((1*time.Second).Nanoseconds()-1),
		date.Location()), nil
}

// getColorScheme constructs a color scheme with spectra going from shades
// of grey to the given color.
func getColorScheme(color color.RGBA) internal.ColorScheme {
	light, _ := colorx.ParseHexColor("#ebedf0")
	dark, _ := colorx.ParseHexColor("#2d333b")
	return internal.ColorScheme{Light: internal.ColorSpectrum{
		Min: light,
		Max: color,
	}, Dark: internal.ColorSpectrum{
		Min: dark,
		Max: color,
	}}
}

func run(cmd *cobra.Command, args []string) error {

	colorStr := viper.GetString(colorCfgKey)
	primaryColor, err := colorx.ParseHexColor(fmt.Sprintf("#%s", colorStr))
	if err != nil {
		return fmt.Errorf("invalid color specification '%s': %w", colorStr, err)
	}

	levels := viper.GetUint(levelsCfgKey)
	if levels < 5 || levels > math.MaxUint8 {
		return fmt.Errorf("invalid number of color levels; allowed range is [5..%d]", math.MaxUint8)
	}

	repositories, err := collectRepositories()
	if err != nil {
		return err
	}
	l := len(repositories)
	var s string
	switch l {
	case 1:
		s = "repository"
	default:
		s = "repositories"
	}
	cmd.Printf("Processing %d %s: %v\n", l, s,
		strings.Join(fp.Map(func(url url.URL) string { return url.String() })(internal.Keys(repositories)), ","))

	lastDay, err := getUntilDate()
	if err != nil {
		return fmt.Errorf("parsing 'until' parameter '%s' failed: %w", lastDay, err)
	}
	logger.Debugw("Analyzing contributions",
		"from", lastDay.AddDate(0, 0, -52*7+1),
		"until", lastDay)

	data := make([]internal.ContributionRecord, 52*7)
	for i := 0; i < 52*7; i++ {
		data[i] = internal.ContributionRecord{
			Date:  lastDay.AddDate(0, 0, -(52*7 - 1 - i)),
			Count: 0,
		}
	}

	if err := addCommitContributions(repositories, lastDay, &data); err != nil {
		return err
	}

	if err := addIssueRelatedContributions(repositories, lastDay, &data); err != nil {
		return err
	}

	if err := addPullRequestReviewRelatedContributions(repositories, lastDay, &data); err != nil {
		return err
	}

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	am := internal.NewContributionMap(data, lastDay, internal.GetColoring(getColorScheme(primaryColor)), uint8(levels))
	err = am.Render(enc)
	if err != nil {
		return fmt.Errorf("rending SVG failed: %w", err)
	}
	err = enc.Flush()
	if err != nil {
		return fmt.Errorf("flushing SVG encoder failed: %w", err)
	}

	filename := viper.GetString(filenameCfgKey)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("can't create output file: %w", err)
	}
	defer f.Close()
	if viper.GetBool(minifyOutputCfgKey) {
		cmd.Printf("Minifying output\n")
		m := minify.New()
		m.AddFunc("image/svg+xml", svg.Minify)
		if err := m.Minify("image/svg+xml", f, &buf); err != nil {
			return fmt.Errorf("output minification failed: %w", err)
		}
	} else {
		_, err := io.Copy(f, &buf)
		if err != nil {
			return fmt.Errorf("writing SVG to file failed: %w", err)
		}
	}
	cmd.Printf("Contribution graph written to '%s'\n", filename)

	return nil
}

// addCommitContributions collects commits from the given repositories into the given contribution records.
func addCommitContributions(repositories map[url.URL]*github.Repository, lastDay time.Time, records *[]internal.ContributionRecord) error {
	for url, repository := range repositories {
		logger.Debugw("Analyzing commit history", "repository", url.String())
		if err := addCommitContributionsForRepo(repository, lastDay, records); err != nil {
			return err
		}
	}
	return nil
}

// addCommitContributionsForRepo collects commits from the given repository into the given contribution records.
func addCommitContributionsForRepo(repository *github.Repository, lastDay time.Time, records *[]internal.ContributionRecord) error {

	var auth *http.BasicAuth
	if viper.IsSet(gitHubTokenCfgKey) {
		auth = &http.BasicAuth{
			Username: "ignore",
			Password: viper.GetString(gitHubTokenCfgKey),
		}
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:  *repository.CloneURL,
		Auth: auth,
	})
	if err != nil {
		return err
	}

	ref, err := r.Head()
	if err != nil {
		return err
	}

	since := lastDay.AddDate(0, 0, -52*7)
	until := lastDay
	commits, err := r.Log(&git.LogOptions{From: ref.Hash(), Since: &since, Until: &until})
	if err != nil {
		return err
	}

	// Parse commit filters
	rawFilters := viper.GetStringSlice(commitFiltersCfgKey)
	var filters []*vm.Program
	for _, fs := range rawFilters {
		filter, err := expr.Compile(fs, expr.Env(object.Commit{}), expr.AsBool())
		if err != nil {
			return fmt.Errorf("invalid commit filter '%s': %w", fs, err)
		}
		filters = append(filters, filter)
	}
	if len(filters) != 0 {
		logger.Debugw("Applying commit filters", "filters", rawFilters)
	}

	filteredCnt := 0
	err = commits.ForEach(func(c *object.Commit) error {

		// Apply commit filters
		filtered := false
		for _, filter := range filters {
			result, err := expr.Run(filter, *c)
			if err != nil {
				return fmt.Errorf("failed to apply filter '%v': %w", filter, err)
			}
			if result.(bool) {
				filtered = true
				break
			}
		}

		if !filtered {
			i := 52*7 - 1 - internal.DaysBetween(c.Committer.When, lastDay)
			(*records)[i].Count++
		} else {
			filteredCnt++
		}
		return nil
	})
	if err != nil {
		return err
	}
	logger.Debugw("Filtered commits", "count", filteredCnt)

	return nil
}

// addIssueRelatedContributions adds opened issues and PRs to the contribution records.
func addIssueRelatedContributions(repositories map[url.URL]*github.Repository, lastDay time.Time, records *[]internal.ContributionRecord) error {
	ctx := context.Background()
	client := github.NewClient(getHTTPClient())
	for _, repository := range repositories {
		owner := repository.GetOwner().GetLogin()
		repo := repository.GetName()
		opt := &github.IssueListByRepoOptions{
			Since:       lastDay.AddDate(0, 0, -52*7),
			State:       "all",
			ListOptions: github.ListOptions{PerPage: 100},
		}
		var allIssues []*github.Issue
		for {
			issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opt)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("fetching issues for repo %s/%s failed (Statuscode: %d)", owner, repo, resp.StatusCode)
			}
			allIssues = append(allIssues, issues...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
		for _, issue := range allIssues {
			idx := 52*7 - 1 - internal.DaysBetween(issue.CreatedAt.Time, lastDay)
			if idx < 0 {
				continue
			}
			(*records)[idx].Count++
		}
	}
	return nil
}

// addPullRequestReviewRelatedContributions adds submitted PR reviews to the contribution records.
// TODO Extract constants, higher-order function for traversal of paginated requests, split into multiple methods (?), make context a parameter, tests
func addPullRequestReviewRelatedContributions(repositories map[url.URL]*github.Repository, lastDay time.Time, records *[]internal.ContributionRecord) error {
	ctx := context.Background()
	client := github.NewClient(getHTTPClient())
	var numberOfPrs, numberOfReviews, numberOfMatchingReviews uint
	for _, repository := range repositories {
		logger.Debugw("Analyzing PR reviews", "repository", repository.CloneURL)
		owner := repository.GetOwner().GetLogin()
		repo := repository.GetName()
		prOpts := &github.PullRequestListOptions{
			State:       "all",
			ListOptions: github.ListOptions{PerPage: 100},
		}
		for {
			prs, resp, err := client.PullRequests.List(ctx, owner, repo, prOpts)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("fetching PRs for repo %s/%s failed (Statuscode: %d)", owner, repo, resp.StatusCode)
			}
			logger.Debugw("Analyzing PRs", "repository", repository.CloneURL, "count", len(prs))
			for _, pr := range prs {
				numberOfPrs++
				reviewOpts := &github.ListOptions{
					PerPage: 100,
				}
				for {
					reviews, listReviewsResp, err := client.PullRequests.ListReviews(ctx, owner, repo, *pr.Number, reviewOpts)
					if err != nil {
						return err
					}
					if listReviewsResp.StatusCode != 200 {
						return fmt.Errorf("fetching reviews for PR #%d of repo %s/%s failed (Statuscode: %d)", pr.Number, owner, repo, resp.StatusCode)
					}
					logger.Debugw("Analyzing PR reviews", "repository", repository.CloneURL, "PR", pr.Number, "count", len(reviews))
					for _, review := range reviews {
						numberOfReviews++
						idx := 52*7 - 1 - internal.DaysBetween(review.SubmittedAt.Time, lastDay)
						match := idx >= 0
						logger.Debugw("PR review processed", "submitted", review.SubmittedAt.Time, "match", match)
						if !match {
							continue
						}
						numberOfMatchingReviews++
						(*records)[idx].Count++
					}
					if listReviewsResp.NextPage == 0 {
						break
					}
					reviewOpts.Page = listReviewsResp.NextPage
				}
			}
			if resp.NextPage == 0 {
				break
			}
			prOpts.Page = resp.NextPage
		}
	}
	logger.Debugw("Finished processing all reviews", "PRs", numberOfPrs, "reviews", numberOfReviews, "matching", numberOfMatchingReviews)
	return nil
}

// Initialize the 'contribution-graph' command.
func init() {
	rootCmd.AddCommand(contributionGraphCmd)

	// Flag to set the last day for which data is visualized
	const untilFlag = "until"
	contributionGraphCmd.Flags().StringP(
		untilFlag,
		"u",
		time.Now().Format("2006-01-02"),
		"Date of last day for which data is visualized")
	if err := viper.BindPFlag(untilCfgKey, contributionGraphCmd.Flags().Lookup(untilFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", untilFlag, "Error", err)
	}

	// Flag to control output minification
	const minifyOutputFlag = "minify"
	contributionGraphCmd.Flags().BoolP(
		minifyOutputFlag,
		"m",
		true,
		"Flag to toggle SVG document minification")
	if err := viper.BindPFlag(minifyOutputCfgKey, contributionGraphCmd.Flags().Lookup(minifyOutputFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", minifyOutputFlag, "Error", err)
	}

	// Flag to control the primary cell color
	const colorFlag = "color"
	contributionGraphCmd.Flags().String(
		colorFlag,
		"39D352",
		"The primary color used for coloring daily contribution cells")
	if err := viper.BindPFlag(colorCfgKey, contributionGraphCmd.Flags().Lookup(colorFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", colorFlag, "Error", err)
	}

	// Flag to control the number of color levels used
	const levelsFlag = "levels"
	contributionGraphCmd.Flags().Uint8P(
		levelsFlag,
		"l",
		5,
		"The number of color levels used for coloring contribution cells")
	if err := viper.BindPFlag(levelsCfgKey, contributionGraphCmd.Flags().Lookup(levelsFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", levelsFlag, "Error", err)
	}

	// Flag to control commit filters used to exclude them from the contributions
	const commitFiltersFlag = "commit-filters"
	contributionGraphCmd.Flags().StringSlice(
		commitFiltersFlag,
		[]string{},
		"Filters used to exclude commits")
	if err := viper.BindPFlag(commitFiltersCfgKey, contributionGraphCmd.Flags().Lookup(commitFiltersFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", commitFiltersFlag, "Error", err)
	}

	const outputFilenameFlag = "output-filename"
	contributionGraphCmd.Flags().StringP(
		outputFilenameFlag,
		"o",
		"contribution-graph.svg",
		"The name of the generated SVG file")
	if err := viper.BindPFlag(filenameCfgKey, contributionGraphCmd.Flags().Lookup(outputFilenameFlag)); err != nil {
		logger.Fatalw("Can't bind to flag", "Flag", outputFilenameFlag, "Error", err)
	}
}
