package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"
)

//CopyFile copies file from src to dst
func CopyFile(src, dst string, closeCloser func(io.Closer)) error {
	if closeCloser == nil {
		closeCloser = func(_ io.Closer) {}
	}
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	rdr, err := os.Open(src) //nolint:gosec
	if err != nil {
		return err
	}
	defer closeCloser(rdr)

	writer, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	if err != nil {
		return err
	}
	defer closeCloser(writer)

	_, err = io.Copy(writer, rdr)
	return err
}

//copyStringMap returns a copy of mp
func copyStringMap(mp map[string]string) map[string]string {
	result := make(map[string]string, len(mp))
	for k, v := range mp {
		result[k] = v
	}
	return result
}

//setStringMapDefault sets map[key] to val unless it is already set
func setStringMapDefault(mp map[string]string, key, val string) {
	_, ok := mp[key]
	if ok {
		return
	}
	mp[key] = val
}

//ExecuteTemplate executes a template
func ExecuteTemplate(tmplString string, os, arch string, vars map[string]string) (string, error) {
	vars = copyStringMap(vars)
	setStringMapDefault(vars, "os", os)
	setStringMapDefault(vars, "arch", arch)
	tmpl, err := template.New("").Option("missingkey=error").Parse(tmplString)
	if err != nil {
		fmt.Println(err.Error())
		return "", fmt.Errorf("%q is not a valid template", tmplString)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		return "", fmt.Errorf("error applying template: %v", err)
	}
	return buf.String(), nil
}
