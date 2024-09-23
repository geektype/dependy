package domain

// Global configuration attributes
type GlobalConfig struct {
	DebugLevel         string   // Debug Level
	DependencyManagers []string // List of names of dependency managers to enable
	DefaultPolicy      string   // Default update policy use when one is not specified by the repository
	RemoteGitProvider  string   // Name of the remote GIT provider
	TitlePrefix        string   // Prefix to use for merge request titles
	RemoveSourceBranch bool     // Whether to delete the dependy branch after successfully merging to main branch
	RunInterval        int
	SquashCommits      bool // Whether to squash all commits of the dependy branch before merging
	FilterTag          string
}
