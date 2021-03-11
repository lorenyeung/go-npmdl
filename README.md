# pkgdl go script

## Purpose
Download a bunch of packages and cache them in a Binary Manager's remote repository

## Installation
Standalone Binary
See the new releases section :) 

Those can be run with ./pkgdl-<DISTRO>-<ARCH>

Source code method
Find your go home (`go env`) 

then install under `$GO_HOME/src` (do not create another folder)
`$ git clone https://github.com/lorenyeung/go-pkgdl.git`

then run
`$ go run $GO_HOME/src/go-pkgdl/pkgdl/pkgdl.go`

Happy downloading! :)

## Usage
### Commands
* apikey
    	API key or password
* credsfile
    	File/Filepath with creds. If there is more than one, it will pick randomly per request. Use whitespace to separate out user and password
* ducheck
    	Disk Usage check in minutes (default 5)
* duthreshold
    	Set Disk usage threshold in % (default 85)
* duwarn
    	Set Disk usage warning in % (default 70)
* forcerepotype
    	force a specific repo type rather than retrieving it from the repository configuration
* log
    	Log level. Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC (default "INFO")
* npmMD
    	Only download NPM Metadata
* queuemax
    	Max queue size before sleeping (default 75)
* random
    	Attempt to pull packages in random queue order
* repo (required)
    	Download Repository
* reset
    	Reset creds file
* uapikey
    	Upstream repository API key or password
* url
    	Binary Manager URL
* user
    	Username
* uuser
    	Upstream repository Username
* v	
        Print the current version and exit
* values
    	Output stored values
* workers
    	Number of workers (default 50)
* workersleep
    	Worker sleep period in seconds (default 5)

## Dependencies
```
golang.org/x/crypto/ssh/terminal
```
