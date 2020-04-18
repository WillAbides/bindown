
## Using bootstrap-bindown.sh

This is the preferred method for ci or development environments. Each release
contains a shell script `bootstrap-bindown.sh` that will download bindown for
the current os. Place `bootstrap-bindown.sh` from the
[latest release](https://github.com/WillAbides/bindown/releases/latest) in your
project's repository. Don't forget to make it executable first (`chmod +x
bootstrap-bindown.sh` on most systems). Then you can call `bootstrap-bindown.sh`
before `bindown` in the projects bootstrap script or Makefile.

### Usage
```
./bootstrap-bindown.sh -h
./bootstrap-bindown.sh: download the bindown binary

Usage: ./bootstrap-bindown.sh [-b bindir] [-d]
  -b sets bindir or installation directory, Defaults to ./bin
  -d turns on debug logging
```

## Go get

If you happen to already have go installed on your system, you can install
bindown with:

```
GO111MODULE=on go get -u github.com/willabides/bindown/v3/cmd/bindown 
```

Note the lowercase `willabides`. Pro tip: Your life will be easier with a
lowercase GitHub username.
