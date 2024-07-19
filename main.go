package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
		return nil, fmt.Errorf("policy %s not found", config.DefaultPolicy)
	}
}

func NewRemoteHandler(global domain.GlobalConfig) (domain.RemoteHandler, error) {
	switch global.RemoteGitProvider {
	case "Gitlab":
		c := viper.Sub("Gitlab")
		if c == nil {
			return nil, errors.New("config for Gitlab not found")
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
		return nil, fmt.Errorf("git provider %s not found", global.RemoteGitProvider)
	}
}

func processRepo(g Global, repo domain.Repository) {
	// TODO: Handle all panics
	slog.Info(fmt.Sprintf("Processing %s repository", repo.Name))
	// Check if a dependy PR already exists
	slog.Debug("Checking if a dependy merge request is already active")

	exists, err := g.remoteHandler.CheckMRExists(repo)
	if err != nil {
		slog.Error("Failed to check if an actie MR exists. Skipping...")
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

	depManager := newManager()

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

func hookHandler(w http.ResponseWriter, _ *http.Request) {
	slog.Info("Hook received")

	_, err := io.WriteString(w, "Pong")
	if err != nil {
		slog.Error("error", slog.Any("error", err))
	}
}

func CheckerFlow(g Global) {
	var wg sync.WaitGroup

	repos, err := g.remoteHandler.GetRepositories()
	if err != nil {
		panic(err)
	}

	for _, r := range repos {
		wg.Add(1)

		go func(r domain.Repository) {
			processRepo(g, r)
			wg.Done()
		}(r)
	}

	wg.Wait()
}

type Global struct {
	gitConfig     GitConfig
	remoteHandler domain.RemoteHandler
	updatePolicy  domain.Policy
}

func main() {
	startMs := time.Now()

	lvlDefault := new(slog.LevelVar)
	lvlDefault.Set(slog.LevelInfo)
	logOpts := &tint.Options{
		Level: lvlDefault,
	}
	logger := slog.New(tint.NewHandler(os.Stdout, logOpts))
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

	switch global.DebugLevel {
	case "DEBUG":
		slog.Info("Using DEBUG log level")
		lvlDefault.Set(slog.LevelDebug)
	case "INFO":
		slog.Info("Using INFO log level")
		lvlDefault.Set(slog.LevelInfo)
	case "WARN":
		lvlDefault.Set(slog.LevelWarn)
	case "ERROR":
		lvlDefault.Set(slog.LevelError)
	case "":
		slog.Info("Defaulting to INFO log level")
	default:
		slog.Error(
			fmt.Sprintf(
				"%s not recongnised as a valid log level. Defaulting to INFO",
				global.DebugLevel,
			),
		)
	}

	updatePolicy, err := newPolicy(global)
	if err != nil {
		slog.Error("Could not initialise update policy", slog.Any("error", err))
		panic(err)
	}

	slog.Info("Update policy set to: " + updatePolicy.GetName())

	remoteHandler, err := NewRemoteHandler(global)
	if err != nil {
		slog.Error("Could not initialise Remote Git Handler", slog.Any("error", err))
		panic(err)
	}

	g := &Global{
		gitConfig:     gitConfig,
		remoteHandler: remoteHandler,
		updatePolicy:  updatePolicy,
	}

	slog.Info("Successfully setup " + remoteHandler.GetName())

	var procWG sync.WaitGroup

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(30000 * time.Millisecond)
	tickDone := make(chan bool)
	serverDone := make(chan bool)
	firstCheck := make(chan bool)

	procWG.Add(1)
	slog.Info("Starting Webhook Server")

	go func() {
		defer procWG.Done()

		server := http.Server{
			Addr:              ":8080",
			ReadHeaderTimeout: 3 * time.Second,
		}

		http.HandleFunc("/", hookHandler)

		go func() {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				slog.Error("Web Server Error", slog.Any("error", err))
			}
		}()
		<-serverDone

		slog.Info("Shutting down webhook server")

		if err := server.Shutdown(context.Background()); err != nil {
			slog.Error("Failed to gracefully shutdown webhook server", slog.Any("error", err))
		}
	}()

	procWG.Add(1)
	slog.Info("Starting Checker")

	go func() {
		defer procWG.Done()

		for {
			select {
			case <-tickDone:
				return
			case <-firstCheck:
				slog.Debug("Initial Check")
				CheckerFlow(*g)
			case t := <-ticker.C:
				fmt.Println("Tick at ", t)
				CheckerFlow(*g)
			}
		}
	}()

	slog.Info(fmt.Sprintf("Finished startup in %s", time.Since(startMs)))

	firstCheck <- true
	// Wait for SIGINT//SIGTERM
	<-sigs

	// Signal goroutines to wind it up
	slog.Info("Attempting to shutdown gracefully")
	ticker.Stop()
	tickDone <- true
	serverDone <- true

	procWG.Wait()
}
