package domain

// Generate candidate dependencies for update
//
// A given policy implementing this interface will decide which dependencies
// to update. The exact strategy for choosing the new candidate will depend
// on each policy's objective. The policy can utilise mechanisms provided by
// a `DependencyManger` to obtain information about a dependency.
type Policy interface {
	// Takes in a list of dependecies and an associated manager and returns a
	// *new* list of dependencies consisting only of the dependencies to be
	// changed.
	GetNextDependencies(current []Dependency, manager DependencyManager) ([]Dependency, error)
}
