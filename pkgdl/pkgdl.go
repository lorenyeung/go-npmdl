package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/debian"
	"go-pkgdl/helpers"
	"go-pkgdl/npm"
	"log"
	"os"
	"os/user"
	"strings"
	"sync"
)

func main() {

	supportedTypes := [3]string{"debian", "npm", "maven"}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configFolder := "/.lorenygo/pkgDownloader/"
	configPath := usr.HomeDir + configFolder

	for i := 0; i < len(supportedTypes); i++ {
		log.Println("Checking that", supportedTypes[i], "downloads folder exists")
		if _, err := os.Stat(configPath + supportedTypes[i] + "Downloads/"); os.IsNotExist(err) {
			log.Println("No config folder found")
			err = os.MkdirAll(configPath+supportedTypes[i]+"Downloads/", 0700)
			helpers.Check(err, true, "Generating "+configPath+" directory")
		} else {
			log.Println(supportedTypes[i], "downloads folder exists, continuing..")
		}
	}
	//TODO clean up downloads dir beforehand

	masterKey := auth.VerifyMasterKey(configPath + "master.key")
	creds := auth.GetDownloadJSON(configPath+"download.json", masterKey)

	var workersVar int
	var usernameVar, apikeyVar, urlVar, repoVar, repoTypeVar string
	var resetVar bool
	flag.IntVar(&workersVar, "workers", 50, "Number of workers")
	flag.StringVar(&usernameVar, "user", "", "Username")
	flag.StringVar(&apikeyVar, "apikey", "", "API key")
	flag.StringVar(&urlVar, "url", creds.URL, "URL")
	flag.StringVar(&repoVar, "repo", "", "Download Repository")
	flag.BoolVar(&resetVar, "reset", false, "Reset creds file")
	flag.StringVar(&repoTypeVar, "pkg", "", "Package type")
	flag.Parse()

	if usernameVar == "" {
		usernameVar = creds.Username
	}
	if apikeyVar == "" {
		apikeyVar = creds.Apikey
	}

	if (repoTypeVar == "" || repoVar == "") && resetVar != true {
		log.Println("Must specify -pkg and -repo")
		os.Exit(0)
	}

	if resetVar == true {
		creds = auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
		usernameVar = creds.Username
		apikeyVar = creds.Apikey
		urlVar = creds.URL
		repoVar = creds.Repository
	}
	if !auth.VerifyAPIKey(urlVar, usernameVar, apikeyVar) {
		if creds.Username == usernameVar && creds.Apikey == apikeyVar && creds.URL == urlVar {
			log.Println("Looks like there's an issue with your credentials file. Reseting")
			auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
			creds = auth.GetDownloadJSON(configPath+"download.json", masterKey)
		} else {
			log.Println("Looks like there's an issue with your custom credentials. Exiting")
			os.Exit(1)
		}
	}

	//update custom
	creds.Username = usernameVar
	creds.Apikey = apikeyVar
	creds.URL = urlVar
	creds.Repository = repoVar

	checkTypeAndRepoParams(creds)

	pkgRepoDlFolder := repoTypeVar + "Downloads"

	//case switch for different package types
	switch repoTypeVar {
	case "debian":
		url := "http://archive.ubuntu.com/ubuntu"
		debian.GetDebianHrefs(url+"/pool/", url, creds.URL, creds.Repository, configPath, creds, 1, "", pkgRepoDlFolder, workersVar)
	case "maven":
		url := "https://jcenter.bintray.com"
		fmt.Println(url)
	case "npm":
		npm.GetNPMJSONList(configPath)
		npm.GetNPMList(configPath)
		file, err := os.Open(configPath + "all-npm-id.txt")
		helpers.Check(err, true, "npm id read")
		defer file.Close()
		scanner := bufio.NewScanner(file)

		//var mutex = &sync.Mutex{} //should help with the concurrent map writes issue
		var ch = make(chan []string, workersVar+1)
		var wg sync.WaitGroup //multi threading the GET details request
		wg.Add(workersVar)
		for i := 0; i < workersVar; i++ {
			go func(i int) {
				for {
					s, ok := <-ch
					if !ok { // if there is nothing to do and the channel has been closed then end the goroutine
						wg.Done()
						return
					}
					npm.GetNPMMetadata(creds, creds.URL+"/api/npm/"+creds.Repository+"/", s[0], s[1], configPath, pkgRepoDlFolder)
				}
			}(i)
		}

		// Now the jobs can be added to the channel, which is used as a queue
		for scanner.Scan() {
			s := strings.Fields(scanner.Text())
			ch <- s
		}
		close(ch) // This tells the goroutines there's nothing else to do
		wg.Wait() // Wait for the threads to finish
	default:
		log.Println("Unsupported package type", repoTypeVar, ". We currently support the following:", supportedTypes)
	}
}

//Test if remote repository exists and is a remote
func checkTypeAndRepoParams(creds auth.Creds) {
	repoCheckData, repoStatusCode := auth.GetRestAPI("GET", true, creds.URL+"/api/repositories/"+creds.Repository, creds.Username, creds.Apikey, "")
	if repoStatusCode != 200 {
		log.Println("Repo", creds.Repository, "does not exist.")
		os.Exit(0)
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(repoCheckData), &result)
	if result["rclass"] != "remote" {
		log.Println(creds.Repository, "is a", result["rclass"], "repository and not a remote repository.")
		os.Exit(0)
	}
}
