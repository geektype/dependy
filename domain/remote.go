package domain

type Repository struct {
	Id     string
	Name   string
	Url    string
	Branch string
}

type RemoteHandler interface {
	CreateMergeRequest(repo Repository, sourceBranch string, targetBranch string) error
}
