package main

import (
	"container/list"
	"encoding/json"
	"flag"
	"go-pkgdl/auth"
	"go-pkgdl/debian"
	"go-pkgdl/docker"
	"go-pkgdl/helpers"
	"go-pkgdl/maven"
	"go-pkgdl/npm"
	"go-pkgdl/pypi"
	"go-pkgdl/rpm"

	"log"
	"os"
	"os/user"
	"sync"
	"time"
)

func main() {

	supportedTypes := [6]string{"debian", "docker", "maven", "npm", "pypi", "rpm"}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configFolder := "/.lorenygo/pkgDownloader/"
	configPath := usr.HomeDir + configFolder

	log.Println("Checking existence of download folders for:", supportedTypes)
	for i := 0; i < len(supportedTypes); i++ {
		if _, err := os.Stat(configPath + supportedTypes[i] + "Downloads/"); os.IsNotExist(err) {
			log.Println("No config folder found")
			err = os.MkdirAll(configPath+supportedTypes[i]+"Downloads/", 0700)
			helpers.Check(err, true, "Generating "+configPath+" directory")
		} else {
		}
	}
	log.Println("Done checking existence for:", supportedTypes)
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
			log.Println("Looks like there's an issue with your credentials file. Resetting")
			auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
			creds = auth.GetDownloadJSON(configPath+"download.json", masterKey)
			usernameVar = creds.Username
			apikeyVar = creds.Apikey
			urlVar = creds.URL
			repoVar = creds.Repository

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
	workQueue := list.New()
	switch repoTypeVar {
	case "debian":
		url := "http://archive.ubuntu.com/ubuntu"
		go func() {
			//func GetDebianHrefs(url string, base string, index int, component string, debianWorkerQueue *list.List) string {
			debian.GetDebianHrefs(url+"/pool/", url, 1, "", workQueue)
		}()

	case "docker":
		url := "https://registry-1.docker.io"
		go func() {
			docker.GetDockerImages(url+"/pool/", url, 1, "", workQueue)
		}()

	case "maven":
		url := "https://jcenter.bintray.com"
		go func() {
			maven.GetMavenHrefs(url+"/", url, workQueue)
		}()

	case "npm":
		npm.GetNPMList(configPath, workQueue)

	case "pypi":
		go func() {
			registry := "https://pypi.org"
			url := "https://files.pythonhosted.org"
			pypi.GetPypiHrefs(registry+"/simple/", registry, url, workQueue)
		}()

	case "rpm":
		go func() {
			log.Print("rpm takes 10 seconds to init, please be patient")
			url := "http://mirror.centos.org"
			rpm.GetRpmHrefs(url+"/centos/", url, workQueue)
		}()

	default:
		log.Println("Unsupported package type", repoTypeVar, ". We currently support the following:", supportedTypes)
	}

	//work queue
	var ch = make(chan interface{}, workersVar+1)
	var wg sync.WaitGroup
	for i := 0; i < workersVar; i++ {
		go func(i int) {
			for {
				s, ok := <-ch
				if !ok {
					wg.Done()
					return
				}
				switch repoTypeVar {
				case "debian":
					md := s.(debian.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder)
					auth.GetRestAPI("PUT", true, creds.URL+"/api/storage/"+creds.Repository+"-cache"+md.URL+"?properties=deb.component="+md.Component+";deb.architecture="+md.Architecture+";deb.distribution="+md.Distribution, creds.Username, creds.Apikey, "")

				case "maven":
					md := s.(maven.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder)

				case "npm":
					md := s.(npm.Metadata)
					npm.GetNPMMetadata(creds, creds.URL+"/api/npm/"+creds.Repository+"/", md.ID, md.Package, configPath, pkgRepoDlFolder)

				case "pypi":
					md := s.(pypi.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder)

				case "rpm":
					md := s.(rpm.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder)
				}
			}
		}(i)
	}
	for {
		for workQueue.Len() == 0 {
			log.Println(repoTypeVar, "work queue is empty, sleeping for 5 seconds...")
			time.Sleep(5 * time.Second)
		}
		s := workQueue.Front().Value
		workQueue.Remove(workQueue.Front())
		ch <- s
	}
	close(ch)
	wg.Wait()

}

func standardDownload(creds auth.Creds, dlURL string, file string, configPath string, pkgRepoDlFolder string) {
	_, headStatusCode := auth.GetRestAPI("HEAD", true, creds.URL+"/"+creds.Repository+"-cache/"+dlURL, creds.Username, creds.Apikey, "")
	if headStatusCode == 200 {
		log.Printf("skipping, got 200 on HEAD request for %s\n", creds.URL+"/"+creds.Repository+"-cache/"+dlURL)
		return
	}

	log.Println("Downloading", creds.URL+"/"+creds.Repository+dlURL)
	auth.GetRestAPI("GET", false, creds.URL+"/"+creds.Repository+dlURL, creds.Username, creds.Apikey, configPath+pkgRepoDlFolder+"/"+file)
	os.Remove(configPath + pkgRepoDlFolder + "/" + file)

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
