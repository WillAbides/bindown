# bindown

**bindown** is a command-line utility to download and install binary dependencies. It is intended to be used in
development and ci environments where it is important to guarantee the same version of the same binary is downloaded
every time.

## Installation

### Using bootstrap-bindown.sh

This is the preferred method for ci or development environments. Each release contains a shell
script `bootstrap-bindown.sh` that will download bindown for the current os. Place `bootstrap-bindown.sh` from the
[latest release](https://github.com/WillAbides/bindown/releases/latest) in your project's repository. Don't forget to
make it executable first (`chmod +x bootstrap-bindown.sh` on most systems). Then you can call `bootstrap-bindown.sh`
before `bindown` in the project's bootstrap script or Makefile.

```
./bootstrap-bindown.sh -h
./bootstrap-bindown.sh: download the bindown binary

Usage: ./bootstrap-bindown.sh [-b bindir] [-d]
  -b sets bindir or installation directory, Defaults to ./bin
  -d turns on debug logging
```

### go install

If you happen to already have go installed on your system, you can install bindown with:

```
go install github.com/willabides/bindown/v3/cmd/bindown@latest 
```

Note the lowercase `willabides`. Pro tip: Your life will be easier with a lowercase GitHub username.

## Quick Start

### Set up your first dependency

This will show you how to get bindown configured to install `jq` in the development environment for your project.

1. Install `bindown` using one of the methods above and make sure it is in your PATH.
2. (optional) Configure completions for your shell.

```shell
$ `bindown install-completions`
```

3. In your project's root create a bindown configuration file (`.bindown.yaml`)

```shell
$ bindown init
```

4. Add a template source that contains a template for jq. We will use https://github.com/WillAbides/bindown-templates.

```shell
$ bindown template-source add origin https://raw.githubusercontent.com/WillAbides/bindown-templates/main/bindown.yml
```

5. Add the jq dependency. It will prompt you for a version. You can use any version you like. 1.6 is currently the
   latest version of jq, so let's use 1.6

```shell
$ bindown dependency add jq --source origin jq
Please enter a value for required variable "version":	1.6
```

6. Install jq to `bin/jq`.

```shell
$ bindown install jq
installed jq to bin/jq
$ bin/jq --version
jq-1.6
```

7. Add the `.bindown` cache directory to your `.gitignore`. If you aren't already ignoring `bin`, you should add that
   too.

```shell
$ echo '/.bindown/' >> .gitignore
$ echo '/bin/' >> .gitignore
```

8. Commit `.bindown.yaml` and `.gitignore`.

```shell
$ git add .bindown.yaml .gitignore
$ git commit -m 'Add initial bindown configuration'
```

### Integrate with make

1. Add this to your `Makefile`

```makefile
BINDOWN_VERSION := 3.10.0

bin/bootstrap-bindown.sh: Makefile
	@mkdir -p bin
	curl -sL https://github.com/WillAbides/bindown/releases/download/v$(BINDOWN_VERSION)/bootstrap-bindown.sh -o $@
	@chmod +x $@

bin/bindown: bin/bootstrap-bindown.sh
	bin/bootstrap-bindown.sh

bin/jq: bin/bindown
	bin/bindown install jq
```

2. Install jq

```shell
$ make bin/jq
curl -sL https://github.com/WillAbides/bindown/releases/download/v3.10.0/bootstrap-bindown.sh -o bin/bootstrap-bindown.sh
bin/bootstrap-bindown.sh
bin/bootstrap-bindown.sh info installed ./bin/bindown
bin/bindown install jq
installed jq to bin/jq
```

3. Run jq

```shell
$ bin/jq --version
jq-1.6
```

### Integrate with scripts-to-rule-them-all

If you use [scripts-to-rule-them-all](https://github.com/github/scripts-to-rule-them-all), you can create scripts for
each dependency bindown manages.

1. Download bootstrap-bindown.sh from the latest release.

```shell
$ curl -L https://github.com/willabides/bindown/releases/latest/download/bootstrap-bindown.sh -o script/bootstrap-bindown.sh
```

2. Create `script/bindown`

```sh
#!/bin/sh

set -e

CDPATH="" cd -- "$(dirname -- "$(dirname -- "$0")")"
mkdir -p bin
[ -f bin/bindown ] || sh script/bootstrap-bindown.sh 2>/dev/null
exec bin/bindown "$@"
```

3. Create `script/jq`

```sh
#!/bin/sh

set -e

CDPATH="" cd -- "$(dirname -- "$(dirname -- "$0")")"
script/bindown install jq > /dev/null
exec bin/jq "$@"
```

4. Make your scripts executable

```shell
$ chmod +x script/bindown script/jq
```

5. Run jq

```shell
$ script/jq --version
jq-1.6
```

## Config file properties

### cache

The directory where bindown will cache downloads and extracted files. This is relative to the directory where the
configuration file resides. cache paths should always use `/` as a delimiter even on Windows or other operating systems
where the native delimiter isn't `/`.

Defaults to `<path to config file>/.bindown`

### install_directory

The directory that bindown installs files to. This is relative to the directory where the configuration file resides.
install_directory paths should always use `/` as a delimiter even on Windows or other operating systems where the native
delimiter isn't `/`.

Defaults to `<path to config file>/bin`

### dependencies

Dependencies are all the dependencies that bindown can install. It is a map where the key is the dependency's name.

Property        | Description
----------------|----------------
`url`           | The url to download a dependency from.
`archive_path`  | The path in the downloaded archive where the binary is located. Default is `./<dependency name>`.
`bin`           | The name of the binary to be installed. Default is the name of the dependency.
`link`          | Whether to create a symlink to the bin instead of copying it.
`template`      | The name of a template to provide default values for this dependency. See [templates](#templates).
`vars`          | A map of variables that will be interpolated in the `url`, `archive_path` and `bin` values. See [vars](#vars)
`overrides`     | A list of value overrides for certain systems. See [overrides](#overrides)
`substitutions` | Values that will be substituted for one variable. See [substitutions](#substitutions)
`systems`       | A list of systems that this dependency is compatible with in format `os/arch`. For example `linux/amd64` or `darwin/arm64`.

### vars

Vars are key value pairs that are used in constructing `url`, `archive_path` and `bin` values using go templates. If you
aren't familiar with go templates, all you need to know is that to use the value from a variable named "foo", you would
write `{{.foo}}`. Go templates can do more than this, but that's all that is practical for bindown.

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

Substitutions provide replacement values for vars. The primary use case is for projects that don't use the same values
for os and arch.

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

It is a list of overrides that each contain a matcher and a dependency. Dependency properties are the same as described
in [dependencies](#dependencies). Matchers match "os", "arch" or vars. A matcher matches if any of its values match the
config value. Matchers can be semver constraints or strings.

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
      version:
        - ">=1.0.0"
    dependency:
      archive_path: special/path/for/arm
```

## Usage

<!--- everything between the next line and the "end usage output" comment is generated by script/generate-readme --->
<!--- start usage output --->
```
Usage: bindown <command>

Flags:
  -h, --help                 Show context-sensitive help.
      --json                 treat config file as json instead of yaml
      --configfile=STRING    file with bindown config. default is the first one of bindown.yml,
                             bindown.yaml, bindown.json, .bindown.yml, .bindown.yaml or
                             .bindown.json ($BINDOWN_CONFIG_FILE)
      --cache=STRING         directory downloads will be cached ($BINDOWN_CACHE)
  -q, --quiet                suppress output to stdout

Commands:
  download                       download a dependency but don't extract or install it
  extract                        download and extract a dependency but don't install it
  install                        download, extract and install a dependency
  format                         formats the config file
  dependency list                list configured dependencies
  dependency add                 add a template-based dependency
  dependency remove              remove a dependency
  dependency info                info about a dependency
  dependency show-config         show dependency config
  dependency update-vars         update dependency vars
  dependency validate            validate that installs work
  template list                  list templates
  template remove                remove a template
  template update-from-source    update a template from source
  template update-vars           update template vars
  template-source list           list configured template sources
  template-source add            add a template source
  template-source remove         remove a template source
  supported-system list          list supported systems
  supported-system add           add a supported system
  supported-system remove        remove a supported system
  checksums add                  add checksums to the config file
  checksums prune                remove unnecessary checksums from the config file
  init                           create an empty config file
  cache clear                    clear the cache
  version                        show bindown version
  install-completions            install shell completions

Run "bindown <command> --help" for more information on a command.
```
<!--- end usage output --->
