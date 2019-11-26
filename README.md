# bindown

**bindown** is a command-line utility to download, verify and install binary files. It is intended to be used in
development and ci environments where it is important to guarantee the same version of the same binary is downloaded
every time.

## Usage

```
 Usage: bindown <command>

Flags:
  --help                            Show context-sensitive help.
  --configfile="buildtools.json"    file with tool definitions
  --cellar-dir=STRING               directory where downloads will be cached

Commands:
  version

  download <target-file>
    download a bin

  config format
    formats the config file

  config update-checksums <target-file>
    name of the binary to update

  config validate <bin>
    validate that downloads work

Run "bindown <command> --help" for more information on a command.
```
