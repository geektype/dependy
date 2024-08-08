package main

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/geektype/dependy/domain"
	"github.com/geektype/dependy/policy"
	"github.com/geektype/dependy/remote"
	"github.com/spf13/viper"
)

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
