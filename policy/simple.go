package policy

import "github.com/geektype/dependy/domain"

// Simple Update Policy
//
// Compares latest non pre-release version to current version and if newer
// version is behind by 1 minor version, the dependency is updated to latest
// version, including latest patch version.
type SimpleUpdatePolicy struct {
}

func (SimpleUpdatePolicy) GetName() string {
	return "SimplePolicy"
}

func (SimpleUpdatePolicy) GetNextDependencies(current []domain.Dependency, manager domain.DependencyManager) ([]domain.Dependency, error) {

	newDeps := make([]domain.Dependency, 0)

	for _, dep := range current {
		newVer, err := manager.FetchLatestVersion(dep)
		if err != nil {
			return nil, err
		}
		// TODO check if greater than 1 minor version
		if dep.Version.LessThan(&newVer) {
			newDeps = append(newDeps, domain.Dependency{
				Name:    dep.Name,
				Version: newVer,
			})
		}
	}
	return newDeps, nil
}
