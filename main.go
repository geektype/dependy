package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/geektype/dependy/domain"
	"github.com/lmittmann/tint"
	"github.com/spf13/viper"
)

type Global struct {
	gitConfig     GitConfig
	remoteHandler domain.RemoteHandler
	updatePolicy  domain.Policy
}

func Checker(g Global) {
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
	// TODO: Should provide default values instead of panicking
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

	ticker := time.NewTicker(5000 * time.Millisecond)
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
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
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
				Checker(*g)
			case t := <-ticker.C:
				fmt.Println("Tick at ", t)
				Checker(*g)
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
