/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package cmd

import (
	"github.com/google/go-github/v50/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Collecting repositories", func() {

	logger = configureLogger()

	When("the GitHub API call for getting the repository errors", func() {
		It("throws an error", func() {
			repoName := "foo/bar"
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetReposByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mock.WriteError(
							w,
							http.StatusInternalServerError,
							"github went belly up or something",
						)
					}),
				))
			client := github.NewClient(mockedHTTPClient)
			_, err := collectRepositories(client, []string{
				repoName,
			})
			Expect(err).Should(HaveOccurred())
		})
	})

	When("given a single repository identifier", func() {
		It("returns that single repository", func() {
			repoName := "foo/bar"
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposByOwnerByRepo,
					github.Repository{
						Name: &repoName,
					},
				),
			)
			client := github.NewClient(mockedHTTPClient)
			repos, err := collectRepositories(client, []string{
				repoName,
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repos).To(HaveLen(1))
			for _, r := range repos {
				Expect(*r.Name).To(Equal(repoName))
			}
		})
	})

	When("given a malformed identifier", func() {
		It("throws an error", func() {
			repo := "foo/*/invalid"
			client := github.NewClient(nil)
			_, err := collectRepositories(client, []string{
				repo,
			})
			Expect(err).Should(HaveOccurred())
		})
	})

	When("given an owner identifier with no owned repositories", func() {
		It("throws an error", func() {
			owner := "foo"
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetOrgsReposByOrg,
					[]github.Repository{},
				),
			)
			client := github.NewClient(mockedHTTPClient)
			_, err := collectRepositories(client, []string{
				owner,
			})
			Expect(err).Should(HaveOccurred())
		})
	})

	When("the GitHub API call for getting own repositories errors", func() {
		It("throws an error", func() {
			owner := "foo"
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetOrgsReposByOrg,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mock.WriteError(
							w,
							http.StatusInternalServerError,
							"github went belly up or something",
						)
					}),
				))
			client := github.NewClient(mockedHTTPClient)
			_, err := collectRepositories(client, []string{
				owner,
			})
			Expect(err).Should(HaveOccurred())
		})
	})

	When("given an owner identifier with multiple owned repositories", func() {
		It("returns a list of the owned repositories", func() {
			owner := "foo"
			repoNames := []string{
				"bar",
				"baz",
			}
			mockedHTTPClient := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetOrgsReposByOrg,
					[]github.Repository{
						{
							Name:    &repoNames[0],
							HTMLURL: &repoNames[0],
						},
						{
							Name:    &repoNames[1],
							HTMLURL: &repoNames[1],
						},
					},
				),
			)
			client := github.NewClient(mockedHTTPClient)
			repos, err := collectRepositories(client, []string{
				owner,
			})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repos).To(HaveLen(2))
			var found []string
			for _, r := range repos {
				found = append(found, *r.Name)
			}
			Expect(found).To(ConsistOf(repoNames))
		})
	})

})
