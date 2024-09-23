package remote

import (
	"fmt"

	"github.com/geektype/dependy/domain"
	"github.com/xanzy/go-gitlab"
)

type GitlabConfig struct {
	URL       string
	AuthToken string
}

func NewGitlabRemoteHandler(
	globalConfig domain.GlobalConfig,
	gitlabConfig GitlabConfig,
) (*GitlabRemoteHandler, error) {
	gClient, err := gitlab.NewClient(gitlabConfig.AuthToken, gitlab.WithBaseURL(gitlabConfig.URL))
	if err != nil {
		return nil, err
	}

	return &GitlabRemoteHandler{
		GitlabURL:          gitlabConfig.URL,
		AuthToken:          gitlabConfig.AuthToken,
		GitlabClient:       gClient,
		RemoveSourceBranch: globalConfig.RemoveSourceBranch,
		SquashCommits:      globalConfig.SquashCommits,
		Topic:              globalConfig.FilterTag,
	}, nil
}

type GitlabRemoteHandler struct {
	GitlabURL          string
	AuthToken          string
	GitlabClient       *gitlab.Client
	RemoveSourceBranch bool
	SquashCommits      bool
	Topic              string
}

func (g *GitlabRemoteHandler) GetName() string {
	return "GitlabRemoteHandler"
}

func (g *GitlabRemoteHandler) CheckMRExists(repo domain.Repository) (bool, error) {
	opt := &gitlab.ListProjectMergeRequestsOptions{
		State:  gitlab.Ptr("opened"),
		Search: gitlab.Ptr("Dependy"),
	}

	mergeRequests, _, err := g.GitlabClient.MergeRequests.ListProjectMergeRequests(repo.ID, opt)
	if err != nil {
		return false, err
	}

	if len(mergeRequests) > 0 {
		return true, nil
	}

	return false, nil
}

func (g *GitlabRemoteHandler) GetRepositories() ([]domain.Repository, error) {
	topic := g.Topic

	p, _, err := g.GitlabClient.Projects.ListProjects(&gitlab.ListProjectsOptions{Topic: &topic})
	if err != nil {
		return nil, err
	}

	repos := make([]domain.Repository, len(p))
	for i, r := range p {
		repos[i] = domain.Repository{
			ID:     fmt.Sprintf("%d", r.ID),
			Name:   r.PathWithNamespace,
			URL:    r.HTTPURLToRepo,
			Branch: r.DefaultBranch,
		}
	}

	return repos, nil
}

func (g *GitlabRemoteHandler) CreateMergeRequest(
	repo domain.Repository,
	sourceBranch string,
	targetBranch string,
) error {
	mrOpts := &gitlab.CreateMergeRequestOptions{
		Title:              gitlab.Ptr("[Dependy] Dependency Update"),
		SourceBranch:       &sourceBranch,
		TargetBranch:       &targetBranch,
		RemoveSourceBranch: &g.RemoveSourceBranch,
		Squash:             &g.SquashCommits,
	}

	_, _, err := g.GitlabClient.MergeRequests.CreateMergeRequest(repo.ID, mrOpts)
	if err != nil {
		return err
	}

	return nil
}
