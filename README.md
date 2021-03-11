# pkgdl go script

## Purpose
Download a bunch of packages and cache them in a Binary Manager's remote repository

## Installation
Standalone Binary
See the new releases section :) 

Those can be run with `./pkgdl-<DISTRO>-<ARCH>`

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
    - Description:
    	- API key or password

* credsfile
    - Description:
    	- File/Filepath with creds. If there is more than one, it will pick randomly per request. Use whitespace to separate out user and password

* ducheck
    - Description:
    	- Disk Usage check in minutes (default 5)

* duthreshold
    - Description:
    	- Set Disk usage threshold in % (default 85)

* duwarn
    - Description:
    	- Set Disk usage warning in % (default 70)

* forcerepotype
    - Description:
    	- Force a specific repo type rather than retrieving it from the repository configuration

* log
    - Description:
    	- Log level. Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC (default "INFO")

* npmMD
    - Description:
    	- Only download NPM Metadata

* queuemax
    - Description:
    	- Max queue size before sleeping (default 75)

* random
    - Description:
    	- Attempt to pull packages in random queue order

* repo (required)
    - Description:
    	- Download Repository name

* reset
    - Description:
    	- Reset creds file

* uapikey
    - Description:
    	- Upstream repository API key or password

* url
    - Description:
    	- Binary Manager URL

* user
    - Description:
    	- Username

* uuser
    - Description:
    	- Upstream repository Username

* v	
    - Description:
        - Print the current version and exit

* values
    - Description:
    	- Output stored values

* workers
    - Description:
    	- Number of workers (default 50)

* workersleep
    - Description:
    	- Worker sleep period in seconds (default 5)

## Dependencies
```
golang.org/x/crypto/ssh/terminal
```
