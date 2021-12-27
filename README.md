# bindown

**bindown** is a command-line utility to download, verify and install binary files. It is intended to be used in
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

### Go get

If you happen to already have go installed on your system, you can install bindown with:

```
GO111MODULE=on go get -u github.com/willabides/bindown/v3/cmd/bindown 
```

Note the lowercase `willabides`. Pro tip: Your life will be easier with a lowercase GitHub username.

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
  --help                   Show context-sensitive help.
  --json                   write json instead of yaml
  --configfile=STRING      file with bindown config. default is the first one of bindown.yml,
                           bindown.yaml, bindown.json, .bindown.yml, .bindown.yaml or .bindown.json
                           ($BINDOWN_CONFIG_FILE)
  --cache=STRING           directory downloads will be cached ($BINDOWN_CACHE)
  --install-completions    install shell completions

Commands:
  version                        show bindown version
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
  add-checksums                  add checksums to the config file
  validate                       validate that installs work
  init                           create an empty config file

Run "bindown <command> --help" for more information on a command.
```
<!--- end usage output --->
