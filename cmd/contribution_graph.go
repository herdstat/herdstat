/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package cmd

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/repeale/fp-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/svg"
	. "herdstat/internal"
	"html/template"
	"io"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
)

// Configuration keys for the contribution-graph command
const (
	// Whether the output SVG should be minified
	minifyOutputCfgKey = "contribution-graph.minify"
	// The name of the output SVG file
	filenameCfgKey = "contribution-graph.filename"
)

// DbDriverName is the name of the database driver configured to use the
// mergestat extension.
const DbDriverName = "sqlite3_with_extensions"

// Register the database driver that uses the mergestat extension.
func init() {
	sql.Register(DbDriverName,
		&sqlite3.SQLiteDriver{
			Extensions: []string{
				"libmergestat",
			},
		})
}

// contributionGraphCmd represents the contribution-graph command
var contributionGraphCmd = &cobra.Command{
	Use:   "contribution-graph",
	Short: "Generates a GitHub-style heatmap to visualize contributions",
	Args:  cobra.NoArgs,
	RunE:  run,
}

// tmplHelpers provides helpers for template processing
var tmplHelpers = template.FuncMap{

	// last is used to check whether the given index references the last
	// element of the given array.
	"last": func(i int, a interface{}) bool {
		return i == reflect.ValueOf(a).Len()-1
	},
}

// commitCountQueryTmpl is a template to generate a query for the number of
// daily commits over a set of repositories.
//
//goland:noinspection SqlResolve
var commitCountQueryTmpl = template.Must(template.New("query").Funcs(tmplHelpers).Parse(`
SELECT strftime('%Y-%m-%d', author_when) Day, COUNT(*)
FROM (
{{- range $i, $e := .}}
    SELECT *
    FROM commits('{{$e}}')
    {{- if not (last $i $)}}
    UNION
    {{- end}}
{{- end}}
)
GROUP BY Day
ORDER BY Day DESC
LIMIT 365;
`))

// createCommitCountQuery instantiates commitCountQueryTmpl for the given
// repository URLs.
func createCommitCountQuery(repos map[url.URL]bool) (string, error) {
	// Translate URLs into strings
	var s []string
	for key := range repos {
		s = append(s, key.String())
	}

	// Instantiate template
	buf := new(bytes.Buffer)
	err := commitCountQueryTmpl.Execute(buf, s)
	if err != nil {
		return "", err
	}
	query := buf.String()
	logger.Debugw("Commit count query generated", "Repository URLs", s, "Query", query)
	return query, nil
}

func run(cmd *cobra.Command, args []string) error {
	db, err := sql.Open(DbDriverName, ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	urls, err := resolveRepositoryURLs()
	if err != nil {
		return err
	}
	l := len(urls)
	var s string
	switch l {
	case 1:
		s = "repository"
	default:
		s = "repositories"
	}
	cmd.Printf("Processing %d %s: %v\n", l, s,
		strings.Join(fp.Map(func(url url.URL) string { return url.String() })(Keys(urls)), ","))

	query, err := createCommitCountQuery(urls)
	if err != nil {
		return fmt.Errorf("query construction failed: %w", err)
	}

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	lastDay := time.Date(2022, 12, 30, 0, 0, 0, 0, time.UTC)
	data := make([]ContributionRecord, 52*7)
	for i := 0; i < 52*7; i++ {
		data[i] = ContributionRecord{
			Date:  lastDay.AddDate(0, 0, -(52*7 - 1 - i)),
			Count: 0,
		}
	}
	var (
		day     string
		commits int
	)
	for rows.Next() {
		err := rows.Scan(&day, &commits)
		if err != nil {
			return fmt.Errorf("could not read database record: %w", err)
		}
		d, err := time.Parse("2006-01-02", day)
		if err != nil {
			return fmt.Errorf("unrecognized date format '%s': %w", day, err)
		}
		if d.After(lastDay) {
			continue
		}
		i := 52*7 - 1 - DaysBetween(d, lastDay)
		if i < 0 {
			break
		}
		data[i] = ContributionRecord{
			Date:  d,
			Count: commits,
		}
		logger.Debugw("Contribution record created", "Date", d, "Contributions", commits)
	}

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	am := NewContributionMap(data, lastDay)
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

// Initialize the 'contribution-graph' command.
func init() {
	rootCmd.AddCommand(contributionGraphCmd)

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
