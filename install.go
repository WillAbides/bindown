package bindown

import (
	"io/fs"
	"os"
	"path/filepath"
)

func install(dep *builtDependency, fsDir fs.FS, targetPath, extractDir string) (string, error) {
	var binName string
	if dep.BinName != nil {
		binName = *dep.BinName
	}
	if binName == "" {
		binName = dep.name
	}
	archivePath := filepath.FromSlash(binName)
	if dep.ArchivePath != nil {
		archivePath = filepath.FromSlash(*dep.ArchivePath)
	}
	if dep.Link != nil && *dep.Link {
		return targetPath, linkBin(targetPath, extractDir, archivePath)
	}
	if fileExists(targetPath) {
		err := os.RemoveAll(targetPath)
		if err != nil {
			return "", err
		}
	}
	err := os.MkdirAll(filepath.Dir(targetPath), 0o755)
	if err != nil {
		return "", err
	}
	err = copyFile(archivePath, targetPath, &copyFileOpts{
		srcFs: fsDir,
	})
	if err != nil {
		return "", err
	}
	targetStat, err := os.Stat(targetPath)
	if err != nil {
		return "", err
	}
	err = os.Chmod(targetPath, targetStat.Mode()|0o750)
	if err != nil {
		return "", err
	}
	return targetPath, nil
}
