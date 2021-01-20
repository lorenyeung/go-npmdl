#!/bin/bash

if [ $# -eq 0 ]
  then
	exe_name="pkgdl"
  else
	exe_name="$1"
fi

#go test ./... -count=1
CGO_ENABLED=0 go build -o $exe_name -ldflags '-w -extldflags "-static"' pkgdl/pkgdl.go
