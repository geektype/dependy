package domain

type Repository struct {
	Id     string
	Name   string
	Url    string
	Branch string
}

type RemoteHandler interface {
    GetName() string
	CreateMergeRequest(repo Repository, sourceBranch string, targetBranch string) error
}
