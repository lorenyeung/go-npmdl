# pkgdl go script

## Purpose
Download a bunch of packages and cache them in an Binary Manager's remote repository

## Installation
Find your go home (`go env`) 

then install under `$GO_HOME/src` (do not create another folder)

`$ git clone https://github.com/lorenyeung/go-pkgdl.git`

then run

`$ go run $GO_HOME/src/go-pkgdl/pkgdl/pkgdl.go`

!Happy downloading!! :)

Quick install steps for Debian
```
cd $HOME
wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
tar -xzf go1.13.3.linux-amd64.tar.gz
mv go /usr/local/
export GOROOT=/usr/local/go
git clone https://github.com/lorenyeung/go-pkgdl.git
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
go version
mkdir -p $HOME/go/src
mv go-pkgdl/ $HOME/go/src/
cd go/src/go-pkgdl/pkgdl/
go get
go build
go ./pkgdl
```

## Dependencies
```
golang.org/x/crypto/ssh/terminal
```
