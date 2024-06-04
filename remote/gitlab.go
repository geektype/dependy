package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/geektype/dependy/domain"
)

type MergeRequest struct {
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	Title              string `json:"title"`
	RemoveSourceBranch bool   `json:"remove_source_branch"`
	Squash             bool   `json:"squash"`
}

type GitlabConfig struct {
	Url              string
	AuthToken        string
	clientTimeoutSec int
	requestMaxRetry  int
}

func NewGitlabRemoteHandler(globalConfig domain.GlobalConfig, gitlabConfig GitlabConfig) *GitlabRemoteHandler {
	client := http.Client{Timeout: time.Duration(gitlabConfig.clientTimeoutSec) * time.Second}
	return &GitlabRemoteHandler{
		GitlabURl:          gitlabConfig.Url,
		AuthToken:          gitlabConfig.AuthToken,
		HttpClient:         client,
		RemoveSourceBranch: globalConfig.RemoveSourceBranch,
		SquashCommits:      globalConfig.SquashCommits,
	}
}

type GitlabRemoteHandler struct {
	GitlabURl          string
	AuthToken          string
	HttpTimeout        int
	HttpClient         http.Client
	RemoveSourceBranch bool
	SquashCommits      bool
}

func (g *GitlabRemoteHandler) CreateMergeRequest(repo domain.Repository, sourceBranch string, targetBranch string) error {
	mr := MergeRequest{
		SourceBranch:       sourceBranch,
		TargetBranch:       targetBranch,
		Title:              "[Dependy] Dependency Update",
		RemoveSourceBranch: g.RemoveSourceBranch,
		Squash:             g.SquashCommits,
	}

	mr_marshalled, err := json.Marshal(mr)
	if err != nil {
		return err
	}

	mrUrl := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests", g.GitlabURl, repo.Id)
	bearerToken := fmt.Sprintf("Bearer %s", g.AuthToken)

	req, err := http.NewRequest("POST",
		mrUrl,
		bytes.NewReader(mr_marshalled))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerToken)

	response, err := g.HttpClient.Do(req)
	if err != nil {
		return nil
	}

	if response.StatusCode != 201 {
		return errors.New("GitLab API error")
	}

	return nil
}
