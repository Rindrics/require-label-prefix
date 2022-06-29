package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/github"
)

func labelsContainPrefix(labels []github.Label, prefix, separator string) bool {
	for _, label := range labels {
		if strings.HasPrefix(label.GetName(), fmt.Sprintf("%s%s", prefix, separator)) {
			return true
		}
	}

	return false
}

func issuesToModify(issues []*github.Issue, config *configuration) []*github.Issue {
	issuesRequiringChanges := []*github.Issue{}

	for _, issue := range issues {
		// Filter out pull requests, which are returned from the issues API.
		if issue.GetPullRequestLinks() != nil {
			continue
		}

		if config.onlyMilestone && issue.GetMilestone() == nil {
			continue
		}

		if !labelsContainPrefix(issue.Labels, config.labelPrefix, config.labelSeparator) {
			issuesRequiringChanges = append(issuesRequiringChanges, issue)
		}
	}

	return issuesRequiringChanges
}

func processIssues(ctx context.Context, githubClient *github.Client, config *configuration) error {
	issues, _, err := githubClient.Issues.ListByRepo(
		ctx,
		config.repoOwner,
		config.repoName,
		&github.IssueListByRepoOptions{
			State: "open",
		})
	if err != nil {
		return fmt.Errorf("error getting issues: %v", err)
	}

	for _, issue := range issuesToModify(issues, config) {
		fmt.Printf(
			"Issue #%d does not have the required label prefix: \"%s%s\"\n",
			issue.GetNumber(),
			config.labelPrefix,
			config.labelSeparator,
		)

		var comment string
		if config.addLabel {
			githubClient.Issues.AddLabelsToIssue(
				ctx,
				config.repoOwner,
				config.repoName,
				issue.GetNumber(),
				[]string{config.defaultLabel},
			)
			comment = fmt.Sprintf("Added default label `%s`. Please consider re-labeling this issue appropriately.", config.defaultLabel)
		} else {
			comment = fmt.Sprintf(
				"No label with prefix \"%s%s\" found. Please add the appropriate label.",
				config.labelPrefix,
				config.labelSeparator,
			)
		}
		_, _, err := githubClient.Issues.CreateComment(
			ctx,
			config.repoOwner,
			config.repoName,
			issue.GetNumber(),
			&github.IssueComment{Body: &comment},
		)
		if err != nil {
			return fmt.Errorf("error adding comment: %v", err)
		}
	}

	return nil
}