package remote

import (
	"fmt"
	"github.com/geektype/dependy/domain"
	"github.com/xanzy/go-gitlab"
)

type GitlabConfig struct {
	Url              string
	AuthToken        string
	clientTimeoutSec int
	requestMaxRetry  int
}

func NewGitlabRemoteHandler(globalConfig domain.GlobalConfig, gitlabConfig GitlabConfig) (*GitlabRemoteHandler, error) {
	gClient, err := gitlab.NewClient(gitlabConfig.AuthToken)
	if err != nil {
		return nil, err
	}
	return &GitlabRemoteHandler{
		GitlabURl:          gitlabConfig.Url,
		AuthToken:          gitlabConfig.AuthToken,
		GitlabClient:       gClient,
		RemoveSourceBranch: globalConfig.RemoveSourceBranch,
		SquashCommits:      globalConfig.SquashCommits,
	}, nil
}

type GitlabRemoteHandler struct {
	GitlabURl          string
	AuthToken          string
	GitlabClient       *gitlab.Client
	RemoveSourceBranch bool
	SquashCommits      bool
}

func (GitlabRemoteHandler) GetName() string {
	return "GitlabRemoteHandler"
}

func (g *GitlabRemoteHandler) CheckMRExists(repo domain.Repository) (bool, error) {
	opt := &gitlab.ListProjectMergeRequestsOptions{
		State:  gitlab.Ptr("opened"),
		Search: gitlab.Ptr("Dependy"),
	}
	merge_requests, _, err := g.GitlabClient.MergeRequests.ListProjectMergeRequests(repo.Id, opt)
	if err != nil {
		return false, err
	}
	if len(merge_requests) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (g *GitlabRemoteHandler) GetRepositories() ([]domain.Repository, error) {

	topic := "dependy"
	p, _, err := g.GitlabClient.Projects.ListProjects(&gitlab.ListProjectsOptions{Topic: &topic})
	if err != nil {
		return nil, err
	}

	repos := make([]domain.Repository, len(p))
	for i, r := range p {
		repos[i] = domain.Repository{
			Id:     fmt.Sprintf("%d", r.ID),
			Name:   r.PathWithNamespace,
			Url:    r.HTTPURLToRepo,
			Branch: r.DefaultBranch,
		}
	}

	return repos, nil
}

func (g *GitlabRemoteHandler) CreateMergeRequest(repo domain.Repository, sourceBranch string, targetBranch string) error {
	mr_opts := &gitlab.CreateMergeRequestOptions{
		Title:              gitlab.Ptr("[Dependy] Dependency Update"),
		SourceBranch:       &sourceBranch,
		TargetBranch:       &targetBranch,
		RemoveSourceBranch: &g.RemoveSourceBranch,
		Squash:             &g.SquashCommits,
	}
	_, _, err := g.GitlabClient.MergeRequests.CreateMergeRequest(repo.Id, mr_opts)
	if err != nil {
		return err
	}
	return nil
}
