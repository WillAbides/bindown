# bindown

[![godoc](https://godoc.org/github.com/WillAbides/bindown?status.svg)](https://pkg.go.dev/github.com/willabides/bindown/v3)
[![Go Report Card](https://goreportcard.com/badge/github.com/WillAbides/bindown)](https://goreportcard.com/report/github.com/WillAbides/bindown)
[![ci](https://github.com/WillAbides/bindown/workflows/ci/badge.svg)](https://github.com/WillAbides/bindown/actions?query=workflow%3Aci+branch%3Amaster+event%3Apush)

**bindown** is a command-line utility to download, verify and install binary files. It is intended to be used in
development and ci environments where it is important to guarantee the same version of the same binary is downloaded
every time.

## Installation

### Using bootstrap-bindown.sh

This is the preferred method for ci or development environments. Each release
contains a shell script `bootstrap-bindown.sh` that will download bindown for
the current os. Place `bootstrap-bindown.sh` from the
[latest release](https://github.com/WillAbides/bindown/releases/latest) in your
project's repository. Don't forget to make it executable first (`chmod +x
bootstrap-bindown.sh` on most systems). Then you can call `bootstrap-bindown.sh`
before `bindown` in the projects bootstrap script or Makefile.

#### Usage
```
./bootstrap-bindown.sh -h
./bootstrap-bindown.sh: download the bindown binary

Usage: ./bootstrap-bindown.sh [-b bindir] [-d]
  -b sets bindir or installation directory, Defaults to ./bin
  -d turns on debug logging
```

### Go get

If you happen to already have go installed on your system, you can install
bindown with:

```
GO111MODULE=on go get -u github.com/willabides/bindown/v3/cmd/bindown 
```

Note the lowercase `willabides`. Pro tip: Your life will be easier with a
lowercase GitHub username.

## Config file properties

### cache

The directory where bindown will cache downloads and extracted files. This is relative to the directory where
the configuration file resides. cache paths should always use `/` as a delimiter even on Windows or other
operating systems where the native delimiter isn't `/`.

Defaults to `<path to config file>/.bindown`

### install_directory

The directory that bindown installs files to. This is relative to the directory where the configuration file
resides. install_directory paths should always use `/` as a delimiter even on Windows or other operating systems
where the native delimiter isn't `/`.

Defaults to `<path to config file>/bin`

### dependencies

Dependencies are all the dependencies that bindown can install. It is a map where the key is the dependency's name.

 Property       | Description                                                                                       
----------------|----------------
`url`           | The url to download a dependency from.                                                            
`archive_path`  | The path in the downloaded archive where the binary is located. Default is `./<dependency name>`. 
`bin`           | The name of the binary to be installed. Default is the name of the dependency.                    
`link`          | Whether to create a symlink to the bin instead of copying it.                                     
`template`      | The name of a template to provide default values for this dependency. See [templates](#templates).            
`vars`          | A map of variables that will be interpolated in the `url`, `archive_path` and `bin` values. See [vars](#vars)
`overrides`     | A list of value overrides for certain systems. See [overrides](#overrides) 
`substitutions` | Values that will be substituted for one variable. See [substitutions](#substitutions) 

### vars

Vars are key value pairs that are used in constructing `url`, `archive_path` and `bin` values using go templates. If
 you aren't familiar with go templates, all you need to know is that to use the value from a variable named "foo", 
 you would write `{{.foo}}`. Go templates can do more than this, but that's all that is practical for bindown.
  
In addition to variables explicitly defined in `vars`, bindown adds `os` and `arch` variables based on the current
 system.
 
Consider this dependency:

```yaml
myproject:
  url: https://github.com/me/myproject/releases/download/v{{.version}}/myproject_{{.version}}_{{.os}}_{{.arch}}.tar.gz
  archive_path: myproject_{{.version}}_{{.os}}_{{.arch}}/myproject
  vars:
    version: 1.2.3
```

When bindown is run for a linux/amd64 system, it will download from 
`https://github.com/me/myproject/releases/download/v1.2.3/myproject_1.2.3_linux_amd64.tar.gz` and use the archive path 
`myproject_1.2.3_linux_amd64/myproject`

### substitutions

Substitutions provides replacement values for vars. The primary use case is for projects that don't use the same
 values for os and arch. 

```yaml
substitutions:
  arch:
    "386": i386
    amd64: x86_64
  os:
    darwin: Darwin
    linux: Linux
    windows: Windows
```

### templates

Templates provide base values for dependencies. If a dependency has an unset value, the value from its template is used.

For `vars`, the value is initially set to the template's `var` map which is then overridden by values from the
 dependency. `substitutions` is handled similarly.
 
`overrides` concatenated with the template values coming first.

Template configuration is identical to dependencies.

### overrides

Overrides allow you to override values for certain operating systems or system architectures. 

It is a list of overrides that each contain a matcher and a dependency. Dependency properties are the same as
 described in [dependencies](#dependencies). Matchers have two properties: `os` and `arch` that each contain a list
  of values to match. A system matches when it's os or arch value matches one of the values listed. 

```yaml
overrides:
  - matcher:
      os:
        - windows
    dependency:
      vars:
        archivepathsuffix: .exe
  - matcher:
      arch:
        - arm
        - arm64
    dependency:
      archive_path: special/path/for/arm
```

## Usage

```
Usage: bindown <command>

Flags:
  --help                        Show context-sensitive help.
  --install-completions         install shell completions
  --configfile="bindown.yml"    file with bindown config ($BINDOWN_CONFIG_FILE)
  --cache=STRING                directory downloads will be cached ($BINDOWN_CACHE)
  --json                        use json instead of yaml for the config file

Commands:
  version

  download <dependency>
    download a dependency but don't extract or install it

  extract <dependency>
    download and extract a dependency but don't install it

  install <dependency>
    download, extract and install a dependency

  format
    formats the config file

  add-checksums <dependency>
    add checksums to the config file

  validate <dependency>
    validate that installs work

Run "bindown <command> --help" for more information on a command.
```
