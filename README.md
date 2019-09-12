# npmdl go script

## Purpose
Download a bunch of npm packages and cache them in an Artifactory remote repository

## Installation
Find your go home (`go env`) 

then install under `$GO_HOME/src` (do not create another folder)

`$ git clone https://github.com/lorenyeung/go-npmdl.git`

then run

`$ go run $GO_HOME/src/go-npmdl/npmdl/npmdl.go`

!!Happy downloading!! :)

## Dependencies
```
golang.org/x/crypto/ssh/terminal
```
