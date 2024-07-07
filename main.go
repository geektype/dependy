package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/geektype/dependy/dependency"
	"github.com/geektype/dependy/domain"
	"github.com/geektype/dependy/policy"
	"github.com/geektype/dependy/remote"
	"github.com/lmittmann/tint"
	"github.com/spf13/viper"
)

func newManager() domain.DependencyManager {
	// TODO: This should be decided based on content of repo
	return dependency.NewGoLangDependencyManager()
}

func newPolicy(config domain.GlobalConfig) (domain.Policy, error) {
	switch config.DefaultPolicy {
	case "simple":
		return policy.SimpleUpdatePolicy{}, nil
	case "":
		slog.Warn("Policy not defined in config, defaulting to SimplePolicy")
		return policy.SimpleUpdatePolicy{}, nil
	default:
		return nil, errors.New(fmt.Sprintf("Policy %s not found", config.DefaultPolicy))
	}
}

func NewRemoteHandler(global domain.GlobalConfig) (domain.RemoteHandler, error) {
	switch global.RemoteGitProvider {
	case "Gitlab":

		c := viper.Sub("Gitlab")
		if c == nil {
			return nil, errors.New("Config for Gitlab not found")
		}
		var gitlab remote.GitlabConfig
		err := c.Unmarshal(&gitlab)
		if err != nil {
			return nil, err
		}
		handler, err := remote.NewGitlabRemoteHandler(global, gitlab)
		if err != nil {
			return nil, err
		}
		return handler, nil

	default:
		return nil, errors.New(fmt.Sprintf("Git provider %s not found", global.RemoteGitProvider))
	}
}

func processRepo(
	repo domain.Repository,
	git_config GitConfig,
	remote_handler domain.RemoteHandler,
	depManager domain.DependencyManager,
	policy domain.Policy,
) {
	slog.Info(fmt.Sprintf("Processing %s repository", repo.Name))
	// Check if a dependy PR already exists
	slog.Debug("Checking if a dependy merge request is already active")
	exists, err := remote_handler.CheckMRExists(repo)
	if err != nil {
		slog.Error("Failed to check if an actie MR exists. Skipping...")
		return
	}
	if exists {
		slog.Info("There is already an active dependy merge request. Skipping...")
		return
	}
	gitM := NewGitManager(git_config)
	err = gitM.CloneRepo(repo)
	if err != nil {
		slog.Error("Failed to clone "+repo.Url, slog.Any("error", err))
		panic(err)
	}

	err = gitM.BranchMain()
	if err != nil {
		slog.Error("Failed to create fix branch", slog.Any("error", err))
		panic(err)
	}

	f, err := gitM.OpenFile(depManager.GetFileName())

	ds, err := depManager.ParseFile(f)
	if err != nil {
		slog.Error("Error parsing "+depManager.GetFileName(), slog.Any("error", err))
		panic(err)
	}

	updated, err := policy.GetNextDependencies(ds, depManager)
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
		depManager.ApplyDependency(d)
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
	err = remote_handler.CreateMergeRequest(repo, git_config.PatchBranchPrefix, repo.Branch)
	if err != nil {
		slog.Error("Failed to create Merge Request", slog.Any("error", err))
		panic(err)
	}

	slog.Info("Done processing")
}

func main() {
	start_ms := time.Now()
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting dependy")

	slog.Info("Reading configuration file")
	viper.SetConfigFile("./config/main.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("Failed to read configuration file", slog.Any("error", err))
		panic(err)
	}

	var global domain.GlobalConfig
	err = viper.Unmarshal(&global)
	if err != nil {
		slog.Error("Failed to unmarshal global config", slog.Any("error", err))
		panic(err)
	}

	gitSub := viper.Sub("git")
	var gitConfig GitConfig
	err = gitSub.Unmarshal(&gitConfig)
	// TODO: Should provide default values instead of panicing
	if err != nil {
		slog.Error("Could not read Git config", slog.Any("error", err))
		panic(err)
	}

	depManager := newManager()
	slog.Info("Successfully setup " + depManager.GetName())

	policy, err := newPolicy(global)
	if err != nil {
		slog.Error("Could not initialise update policy", slog.Any("error", err))
		panic(err)
	}
	slog.Info("Update policy set to: " + policy.GetName())

	remoteHandler, err := NewRemoteHandler(global)
	if err != nil {
		slog.Error("Could not initialise Remote Git Handler", slog.Any("error", err))
		panic(err)
	}
	slog.Info("Successfully setup " + remoteHandler.GetName())

	slog.Info(fmt.Sprintf("Finished startup in %s", time.Now().Sub(start_ms)))

	slog.Info("Starting Dependy!")

	var wg sync.WaitGroup

	ticker := time.NewTicker(5000 * time.Millisecond)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Tick at ", t)
				repos, err := remoteHandler.GetRepositories()
				if err != nil {
					panic(err)
				}

				for _, r := range repos {
					wg.Add(1)
					go func(r domain.Repository) {
						processRepo(r, gitConfig, remoteHandler, depManager, policy)
						wg.Done()
					}(r)
				}
				wg.Wait()
			}
		}
	}()

	fmt.Scanln()
	slog.Info("Attempting to shutdown gracefully")
	ticker.Stop()
	done <- true
}
