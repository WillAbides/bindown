
## About

**bindown** is a command-line utility to download, verify and install binary files. It is intended to be used in
development and ci environments where it is important to guarantee the same version of the same binary is downloaded
every time.

## Quick Start

#### Install bindown into your project space

To try out bindown, the quickest way to install it is by running the command below. It will download the latest
bindown to `./bin/bindown`. This is not the recommended way to install normally. After all, bindown is about
guaranteeing every environment gets the same version of your dependencies. We will cover how to properly set up a
project later.

```shell
sh -c "$(curl -sfL https://github.com/WillAbides/bindown/releases/latest/download/bootstrap-bindown.sh)"
```

If you are wary of this method, you can manually and extract [the latest release](https://github.com/WillAbides/bindown/releases/latest).
Move the extracted bindown to `./bin/bindown` to make following the quick start guide easier.

To verify that bindown is working:

```shell
$ bin/bindown version
bindown: version 3.3.1
```

#### Create a bindown config

```shell
bin/bindown init
```

This creates `bindown.yml` in the current directory.

#### Add a template source

The fastest way to get up and running with bindown is using existing templates. This step installs a template source.

For this example we will add the bindown_templates.yml file from the bindown repo and call it "origin"

```shell
bin/bindown template-source add origin https://raw.githubusercontent.com/WillAbides/bindown/main/bindown_templates.yml
```

Now let's see what is available in origin:

```shell
$ bin/bindown template list --source origin
go
golangci-lint
goreleaser
jq
mockgen
semver-next
yq
```

#### Add a dependency

Let's make "jq" our first dependency.

We add it to our configuration with the command below.  We will be prompted for a version.  Let's use 1.6.

```shell
$ bin/bindown dependency add jq origin#jq
Please enter a value for required variable "version":	1.6
```

I want to point out that final argument "origin#jq". That is the template to build the dependency from in the format
 of "source#name".

#### Add checksums and validate

Now we have a dependency in our config. Before we can use it we need to add checksums to the configuration and
 validate that installation works on all supported systems.
 
```shell
bin/bindown add-checksums
bin/bindown validate jq
```

Your config should now look something like this
<details><summary>bindown.yml</summary><p>

```yaml
dependencies:
  jq:
    template: origin#jq
    vars:
      version: "1.6"
templates:
  origin#jq:
    url: https://github.com/stedolan/jq/releases/download/jq-{{.version}}/jq-{{.os}}{{.arch}}{{.extension}}
    archive_path: jq-{{.os}}{{.arch}}{{.extension}}
    bin: jq
    vars:
      extension: ""
    required_vars:
    - version
    overrides:
    - matcher:
        os:
        - darwin
        arch:
        - amd64
      dependency:
        url: https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64
        archive_path: jq-osx-amd64
    - matcher:
        os:
        - windows
      dependency:
        vars:
          extension: .exe
    substitutions:
      arch:
        "386": "32"
        amd64: "64"
      os:
        windows: win
    systems:
    - linux/386
    - linux/amd64
    - darwin/amd64
    - windows/386
    - windows/amd64
template_sources:
  origin: https://raw.githubusercontent.com/WillAbides/bindown/main/bindown_templates.yml
url_checksums:
  https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux32: 319af6123aaccb174f768a1a89fb586d471e891ba217fe518f81ef05af51edd9
  https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64: af986793a515d500ab2d35f8d2aecd656e764504b789b66d7e1a0b727a124c44
  https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64: 5c0a0a3ea600f302ee458b30317425dd9632d1ad8882259fcaf4e9b868b2b1ef
  https://github.com/stedolan/jq/releases/download/jq-1.6/jq-win32.exe: 0012cb4c0eb6eaf97b842e676e423a69a8fea95055d93830551b4a5a54494bd8
  https://github.com/stedolan/jq/releases/download/jq-1.6/jq-win64.exe: a51d36968dcbdeabb3142c6f5cf9b401a65dc3a095f3144bd0c118d5bb192753
```
</p></details>

#### Install jq

