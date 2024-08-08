package main

import (
	"fmt"
	"log/slog"

	"github.com/geektype/dependy/dependency"
	"github.com/geektype/dependy/domain"
)

func processRepo(g Global, repo domain.Repository) {
	// TODO: Handle all panics
	slog.Info(fmt.Sprintf("Processing %s repository", repo.Name))
	// Check if a dependy PR already exists
	slog.Debug("Checking if a dependy merge request is already active")

	exists, err := g.remoteHandler.CheckMRExists(repo)
	if err != nil {
		slog.Error("Failed to check if an active MR exists. Skipping...")
		return
	}

	if exists {
		slog.Info("There is already an active dependy merge request. Skipping...")
		return
	}

	gitM := NewGitManager(g.gitConfig)

	err = gitM.CloneRepo(repo)
	if err != nil {
		slog.Error("Failed to clone "+repo.URL, slog.Any("error", err))
		panic(err)
	}

	err = gitM.BranchMain()
	if err != nil {
		slog.Error("Failed to create fix branch", slog.Any("error", err))
		panic(err)
	}

	// TODO: This should be decided based on repo content
	var depManager domain.DependencyManager = dependency.NewGoLangDependencyManager()

	f, err := gitM.OpenFile(depManager.GetFileName())
	if err != nil {
		slog.Error("Error opening file: ", slog.Any("error", err))
	}

	ds, err := depManager.ParseFile(f)
	if err != nil {
		slog.Error("Error parsing "+depManager.GetFileName(), slog.Any("error", err))
		panic(err)
	}

	updated, err := g.updatePolicy.GetNextDependencies(ds, depManager)
	if err != nil {
		slog.Error("Error while fetching latest dependency versions", slog.Any("error", err))
		panic(err)
	}

	if len(updated) == 0 {
		slog.Info("Already up to date. Skipping")
		return
	}

	slog.Info("Updating dependencies")

	for _, d := range updated {
		err := depManager.ApplyDependency(d)
		if err != nil {
			slog.Error("Could not apply dependency update", slog.Any("error", err))
		}
	}

	final, err := depManager.GetFile()
	if err != nil {
		slog.Error("Failed to edit "+depManager.GetFileName(), slog.Any("error", err))
		panic(err)
	}

	err = gitM.CommitFile(depManager.GetFileName(), final)
	if err != nil {
		slog.Error("Error encountered while creating commit", slog.Any("error", err))
		panic(err)
	}

	slog.Info("Pushing changes to remote")

	err = gitM.Push()
	if err != nil {
		slog.Error("Failed to push to remote repository", slog.Any("error", err))
		return
	}

	slog.Info("Creating merge request")

	err = g.remoteHandler.CreateMergeRequest(repo, g.gitConfig.PatchBranchPrefix, repo.Branch)
	if err != nil {
		slog.Error("Failed to create Merge Request", slog.Any("error", err))
		panic(err)
	}

	slog.Info("Done processing")
}
