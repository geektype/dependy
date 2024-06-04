package main

import (
	"github.com/geektype/dependy/dependency"
	"github.com/geektype/dependy/domain"
	"github.com/geektype/dependy/policy"
	"github.com/geektype/dependy/remote"
	"github.com/spf13/viper"
)

func newManager() domain.DependencyManager {
	return dependency.NewGoLangDependencyManager()
}

func newPolicy() domain.Policy {
	return policy.SimpleUpdatePolicy{}
}

func NewRemoteHandler(global domain.GlobalConfig) domain.RemoteHandler {
	c := viper.Sub("gitlab")
	if c == nil {
		panic("Config for Gitlab not found")
	}
	var gitlab remote.GitlabConfig
	err := c.Unmarshal(&gitlab)
	if err != nil {
		panic(err)
	}
	return remote.NewGitlabRemoteHandler(global, gitlab)
}

func main() {
	viper.SetConfigFile("./config/main.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	var global domain.GlobalConfig
	err = viper.Unmarshal(&global)
	if err != nil {
		panic(err)
	}

	manager := newManager()
	policy := newPolicy()
	handler := NewRemoteHandler(global)

	gitSub := viper.Sub("git")
	var gitConfig GitConfig
	err = gitSub.Unmarshal(&gitConfig)
	if err != nil {
		panic(err)
	}
	gitM := NewGitManager(gitConfig)

	repo := domain.Repository{
		Id:     "58150706",
		Name:   "Microservice",
		Url:    "https://gitlab.com/abdullah_ahmed_02/microservice.git",
		Branch: "master",
	}

	err = gitM.CloneRepo(repo)
	if err != nil {
		panic(err)
	}

	err = gitM.BranchMain()
	if err != nil {
		panic(err)
	}

	f, err := gitM.OpenFile(manager.GetFileName())

	ds, err := manager.ParseFile(f)
	if err != nil {
		panic(err)
	}

	updated, err := policy.GetNextDependencies(ds, manager)

	for _, d := range updated {
		manager.ApplyDependency(d)
	}

	final, err := manager.GetFile()
	if err != nil {
		panic(err)
	}

	err = gitM.CommitFile(manager.GetFileName(), final)
	if err != nil {
		panic(err)
	}

	err = gitM.Push()
	if err != nil {
		panic(err)
	}

	err = handler.CreateMergeRequest(repo, gitConfig.PatchBranchPrefix, repo.Branch)
	if err != nil {
		panic(err)
	}

}
