package dependency

import (
	"errors"

	"github.com/Masterminds/semver/v3"
	"github.com/edoardottt/depsdev/pkg/depsdev"
	"github.com/geektype/dependy/domain"
	"golang.org/x/mod/modfile"
)

func NewGoLangDependencyManager() *GoLangDependencyManager {
	depsClient := depsdev.NewAPI()
	return &GoLangDependencyManager{
		ApiClient: depsClient,
	}
}

type GoLangDependencyManager struct {
	RawFile   []byte
	ModFile   *modfile.File
	ApiClient *depsdev.API
}

func (*GoLangDependencyManager) GetName() string {
    return "GoLangManager"
}

func (g *GoLangDependencyManager) GetFileName() string {
	return "go.mod"
}

func (g *GoLangDependencyManager) ParseFile(contents []byte) ([]domain.Dependency, error) {
	g.RawFile = contents
	f, err := modfile.Parse(g.GetFileName(), g.RawFile, nil)
	if err != nil {
		return nil, err
	}
	g.ModFile = f

	newDeps := make([]domain.Dependency, 0)

	for _, req := range g.ModFile.Require {
		var dep domain.Dependency
		if !req.Indirect {
			dep.Name = req.Mod.Path
			v, err := semver.NewVersion(req.Mod.Version)
			if err != nil {
				return nil, err
			}
			dep.Version = *v
			newDeps = append(newDeps, dep)
		}
	}

	return newDeps, nil
}

func (g *GoLangDependencyManager) FetchLatestVersion(dep domain.Dependency) (semver.Version, error) {
	def_ver := semver.New(0, 0, 0, "", "")
	info, err := g.ApiClient.GetInfo("go", dep.Name)
	if err != nil {
		return *def_ver, err
	}

	for i := len(info.Versions) - 1; i >= 0; i-- {
		if info.Versions[i].IsDefault {
			ver, err := semver.NewVersion(info.Versions[i].VersionKey.Version)
			if err != nil {
				return *def_ver, err
			}
			return *ver, nil
		}
	}
	return *def_ver, nil
}

func (g *GoLangDependencyManager) ApplyDependency(dependency domain.Dependency) error {
	for _, r := range g.ModFile.Require {
		if r.Mod.Path == dependency.Name {
			r.Mod.Version = "v" + dependency.Version.String()
			return nil
		}
	}
	return errors.New("Depenedency not found in file")
}

func (g *GoLangDependencyManager) GetFile() ([]byte, error) {
	// Dodgy hack...
	g.ModFile.SetRequire(g.ModFile.Require)
	f, err := g.ModFile.Format()
	if err != nil {
		return nil, err
	}
	return f, nil
}
