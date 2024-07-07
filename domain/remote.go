package domain

// Handler for interacting with a remote GIT provider
//
// Responsible for interacting with relevant interfaces exposed by a GIT
// provider i.e Github. Lean on remote provider's capabilities as much as
// possible when implementing this interface to reduce internal complexity.
type RemoteHandler interface {
	// Provide a unique name to represent the remote provider
	GetName() string

	// Fetch repositories matching given filters and criteria
	//
	// TODO: Implement filters and search criteria
	GetRepositories() ([]Repository, error)

	// Create equivalent of a merge request in remote to merge dependy branch with main branch
	CreateMergeRequest(repo Repository, sourceBranch string, targetBranch string) error

	// Check if there is already an active merge request in the repository
	CheckMRExists(repo Repository) (bool, error)
}

// A Git Repository provided by a remote provider
type Repository struct {
	Id     string // Identifier assigned by remote (not related to GIT)
	Name   string // Human friendly repo name of the format <namespace>/<repo_name>. i.e geektype/dependy
	Url    string // **HTTPS** remote URL for Repository
	Branch string // Name of the designated main branch
}
