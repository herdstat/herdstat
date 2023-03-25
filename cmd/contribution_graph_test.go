/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package cmd

import (
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v50/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/rand"
	"herdstat/internal"
	"net/url"
	"os"
	"time"
)

func createRepository() (*git.Repository, *url.URL, error) {
	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		return nil, nil, err
	}
	cloneUrl, err := url.Parse("file://" + dir)
	if err != nil {
		return nil, nil, err
	}
	r, err := git.PlainInit(dir, false)
	if err != nil {
		return nil, cloneUrl, err
	}
	return r, cloneUrl, nil
}

func signature(when time.Time) *object.Signature {
	return &object.Signature{
		Name:  "Jane Roe",
		Email: "jane.roe@herdstat.com",
		When:  when,
	}
}

func createCommit(r *git.Repository, time time.Time) error {
	name := fmt.Sprint(rand.Uint64())
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	file := w.Filesystem.Root() + "/" + name
	f, err := os.Create(file)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = w.Add(name)
	if err != nil {
		return err
	}
	sig := signature(time)
	_, err = w.Commit("Lorem ipsum", &git.CommitOptions{
		Author:    sig,
		Committer: sig,
	})
	return nil
}

var _ = Describe("Analyzing commits", func() {

	logger = configureLogger()

	When("given a repo with a commit on a specific day", func() {
		It("updates the contribution records accordingly", func() {
			r, url, err := createRepository()
			Expect(err).NotTo(HaveOccurred())
			commitTime := time.Date(2013, time.April, 22, 23, 0, 0, 0, time.UTC)
			err = createCommit(r, commitTime)
			Expect(err).NotTo(HaveOccurred())
			repo := &github.Repository{
				CloneURL: github.String(url.String()),
			}
			lastDay, err := dateparse.ParseStrict("2013-04-22 23:59")
			Expect(err).NotTo(HaveOccurred())
			data := make([]internal.ContributionRecord, 52*7)
			for i := 0; i < 52*7; i++ {
				data[i] = internal.ContributionRecord{
					Date:  lastDay.AddDate(0, 0, -(52*7 - 1 - i)),
					Count: 0,
				}
			}
			err = addCommitContributionsForRepo(repo, lastDay, &data)
			Expect(err).NotTo(HaveOccurred())
			Expect(data[52*7-1].Count).To(Equal(1))
		})
	})
})
