package bindown

import (
	"os"
	"path/filepath"
)

func install(dep *Dependency, targetPath, extractDir string) (string, error) {
	if !dep.built {
		panic("install called on non-built dependency")
	}
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
	extractBin := filepath.Join(extractDir, archivePath)
	if dep.Link != nil && *dep.Link {
		return targetPath, linkBin(targetPath, extractBin)
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
	err = copyFile(extractBin, targetPath)
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
