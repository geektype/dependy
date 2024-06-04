package domain

type GlobalConfig struct {
	DependencyManagers []string
	DefaultPolicy      string
	RemoteGitProvider  string
	TitlePrefix        string
	RemoveSourceBranch bool
	SquashCommits bool
}
