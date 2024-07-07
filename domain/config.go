package domain

// Global configuration attributes
type GlobalConfig struct {
    // List of names of dependency managers to enable
	DependencyManagers []string
    // Default update policy use when one is not specified by the repository
	DefaultPolicy      string
    // Nmae of the remote GIT provider
	RemoteGitProvider  string
    // Prefix to use for merge request titles 
	TitlePrefix        string
    // Whether to delete the dependy branch after succesfully merging to main branch
	RemoveSourceBranch bool
    // Whether to squash all commits of the dependy branch before merging
	SquashCommits      bool
}
