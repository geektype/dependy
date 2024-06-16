package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
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
        return remote.NewGitlabRemoteHandler(global, gitlab), nil

    default:
        return nil, errors.New(fmt.Sprintf("Git provider %s not found", global.RemoteGitProvider))
    }
}

func main(){
    start_ms := time.Now()
    logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug}))
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

	manager := newManager()
    slog.Info("Successfully setup " + manager.GetName())

	policy, err := newPolicy(global)
    if err != nil {
        slog.Error("Could not initialise update policy", slog.Any("error", err))
        panic(err)
    }
    slog.Info("Update policy set to: " + policy.GetName())

	handler, err := NewRemoteHandler(global)
    if err != nil {
        slog.Error("Could not initialise Remote Git Handler", slog.Any("error", err))
        panic(err)
    }
    slog.Info("Successfully setup " + handler.GetName())

	gitM := NewGitManager(gitConfig)

    slog.Info(fmt.Sprintf("Finished startup in %s", time.Now().Sub(start_ms)))

	repo := domain.Repository{
		Id:     "58150706",
		Name:   "Microservice",
		Url:    "https://gitlab.com/abdullah_ahmed_02/microservice.git",
		Branch: "master",
	}

    slog.Info(fmt.Sprintf("Processing %s repository", repo.Name))
	err = gitM.CloneRepo(repo)
	if err != nil {
        slog.Error("Failed to clone " + repo.Url, slog.Any("error", err))
		panic(err)
	}

	err = gitM.BranchMain()
	if err != nil {
        slog.Error("Failed to create fix branch", slog.Any("error", err))
		panic(err)
	}

	f, err := gitM.OpenFile(manager.GetFileName())

	ds, err := manager.ParseFile(f)
	if err != nil {
        slog.Error("Error parsing " + manager.GetFileName(), slog.Any("error", err))
		panic(err)
	}

	updated, err := policy.GetNextDependencies(ds, manager)
    if err != nil {
        slog.Error("Error while fetching latest dependency versions", slog.Any("error", err))
        panic(err)
    }

    if len(updated) == 0 {
        slog.Info("Already up to date. Skipping")
        os.Exit(0)
    }
    slog.Info("Updating dependencies")
	for _, d := range updated {
		manager.ApplyDependency(d)
	}

	final, err := manager.GetFile()
	if err != nil {
        slog.Error("Failed to edit " + manager.GetFileName(), slog.Any("error", err))
		panic(err)
	}

	err = gitM.CommitFile(manager.GetFileName(), final)
	if err != nil {
        slog.Error("Error encountered while creating commit", slog.Any("error", err))
		panic(err)
	}

    slog.Info("Pusing changes to remote")
	err = gitM.Push()
	if err != nil {
        slog.Error("Failed to push to remote repository", slog.Any("error", err))
		panic(err)
	}

    slog.Info("Creating merge request")
	err = handler.CreateMergeRequest(repo, gitConfig.PatchBranchPrefix, repo.Branch)
	if err != nil {
        slog.Error("Failed to create Merge Request", slog.Any("error", err))
		panic(err)
	}

    slog.Info("Done processing")

}
