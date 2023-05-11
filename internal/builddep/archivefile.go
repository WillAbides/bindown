package builddep

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

type archiveFile struct {
	origName    string
	name        string
	suffix      string
	tmplCount   int
	executable  bool
	containsBin bool
}

// archiveFileLess puts executables first,
// then containsBin,
// then the most templated files,
// then the shortest path, then alphabetically
func archiveFileLess(a, b *archiveFile) bool {
	if a.executable != b.executable {
		return a.executable
	}
	if a.containsBin != b.containsBin {
		return a.containsBin
	}
	fTmpls := strings.Count(a.name, "{{")
	otherTmpls := strings.Count(b.name, "{{")
	if fTmpls != otherTmpls {
		return fTmpls > otherTmpls
	}
	fSlashes := strings.Count(a.origName, "/")
	otherSlashes := strings.Count(b.origName, "/")
	if fSlashes != otherSlashes {
		return fSlashes < otherSlashes
	}
	return a.origName < b.origName
}

// archiveFileGroupable returns true if a and b can be in the same top-level dependency
func archiveFileGroupable(a, b *archiveFile) bool {
	return a.name == b.name && a.suffix == b.suffix
}

func parseArchiveFile(origName, binName, osName, archName, version string, executable bool) *archiveFile {
	a := archiveFile{
		origName:    origName,
		name:        origName,
		executable:  executable,
		containsBin: strings.Contains(origName, binName),
	}
	if osName != "" {
		for {
			idx := strings.Index(a.name, osName)
			if idx == -1 {
				break
			}
			a.tmplCount++
			a.name = a.name[:idx] + "{{.os}}" + a.name[idx+len(osName):]
		}
	}
	if archName != "" {
		for {
			idx := strings.Index(a.name, archName)
			if idx == -1 {
				break
			}
			a.tmplCount++
			a.name = a.name[:idx] + "{{.arch}}" + a.name[idx+len(archName):]
		}
	}
	for {
		idx := strings.Index(a.name, version)
		if idx == -1 {
			break
		}
		a.tmplCount++
		a.name = a.name[:idx] + "{{.version}}" + a.name[idx+len(version):]
	}
	// .exe is the only suffix we care about
	if strings.HasSuffix(a.name, ".exe") {
		a.suffix = ".exe"
		a.name = a.name[:len(a.name)-4]
	}
	a.name += "{{.archivePathSuffix}}"
	return &a
}

type archiveFileCandidate struct {
	archiveFile *archiveFile
	matches     []*dlFile
	nonMatches  []*dlFile
}

type selectCandidateFunc func([]*archiveFileCandidate, *archiveFileCandidate) error

func defaultSelectCandidateFunc(candidates []*archiveFileCandidate, candidate *archiveFileCandidate) error {
	options := make([]string, len(candidates))
	optionsMap := map[string]*archiveFileCandidate{}
	for i := range candidates {
		text := fmt.Sprintf("%s - (%s)", candidates[i].archiveFile.name, candidates[i].archiveFile.origName)
		options[i] = text
		optionsMap[text] = candidates[i]
	}
	var choice string
	err := survey.AskOne(&survey.Select{
		Message: "Select the correct archive file",
		Options: options,
	}, &choice)
	if err != nil {
		return err
	}
	*candidate = *optionsMap[choice]
	return nil
}
