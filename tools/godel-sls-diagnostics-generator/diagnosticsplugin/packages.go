package diagnosticsplugin

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/palantir/pkg/matcher"
	"github.com/palantir/pkg/pkgpath"
	"github.com/pkg/errors"
)

// PkgsInProject returns all the packages in the projectDir except those that match exclude.
// Taken from https://github.com/palantir/okgo/blob/d5f6b9f4/cmd/check.go#L61-L96
func PkgsInProject(projectDir string, exclude matcher.Matcher) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine working directory")
	}
	if !filepath.IsAbs(projectDir) {
		projectDir = filepath.Join(wd, projectDir)
	}
	var relPathPrefix string
	if wd != projectDir {
		relPathPrefixVal, err := filepath.Rel(wd, projectDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to determine relative path")
		}
		relPathPrefix = relPathPrefixVal
	}
	pkgs, err := pkgpath.PackagesInDir(projectDir, exclude)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list packages")
	}
	pkgPaths, err := pkgs.Paths(pkgpath.Relative)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get package paths")
	}
	if relPathPrefix != "" {
		for i, pkgPath := range pkgPaths {
			pkgPaths[i] = "./" + path.Join(relPathPrefix, pkgPath)
		}
	}
	for i := range pkgPaths {
		if strings.HasPrefix(pkgPaths[i], "./..") {
			pkgPaths[i] = strings.TrimPrefix(pkgPaths[i], "./")
		}
	}
	return pkgPaths, nil
}
