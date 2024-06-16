package domain

import "github.com/Masterminds/semver/v3"

// Represents abstraciton for ecosystem specific dependency management
//
// Every ecosystem manager (i.e) Go, Cargo, PyPi, NPM etc should implement this
// to handle the parsing and extraction of *relavent* dependencies from
// the given file. As well as providing a mechanism for fetching the latest
// version of a given dependency from an appropriate data source.
type DependencyManager interface {
    // Get the name of the manager
    GetName() string

	// Get the name of the file this manager supports
	GetFileName() string

	// Reads a dependency file and extracts candidate dependencies
	ParseFile(content []byte) ([]Dependency, error)

	// Fetch latest non pre-release version
	FetchLatestVersion(dep Dependency) (semver.Version, error)

	// Replace existing dependencies in file with the one given
	ApplyDependency(dependency Dependency) error

	// Get back a byte representation of the dependency file
	GetFile() ([]byte, error)
}

type Dependency struct {
	Name    string
	Version semver.Version
}
