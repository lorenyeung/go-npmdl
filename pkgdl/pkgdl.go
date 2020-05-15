package main

import (
	"container/list"
	"encoding/json"
	"flag"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/debian"
	"go-pkgdl/docker"
	"go-pkgdl/generic"
	"go-pkgdl/helpers"
	"go-pkgdl/maven"
	"go-pkgdl/npm"
	"go-pkgdl/pypi"
	"go-pkgdl/rpm"
	"os"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
)

func main() {

	var workersVar int
	var usernameVar, apikeyVar, urlVar, repoVar, logLevelVar string
	var resetVar, valuesVar bool
	flag.StringVar(&logLevelVar, "log", "INFO", "Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC")
	flag.IntVar(&workersVar, "workers", 50, "Number of workers")
	flag.StringVar(&usernameVar, "user", "", "Username")
	flag.StringVar(&apikeyVar, "apikey", "", "API key")
	flag.StringVar(&urlVar, "url", "", "URL")
	flag.StringVar(&repoVar, "repo", "", "Download Repository")
	flag.BoolVar(&resetVar, "reset", false, "Reset creds file")
	flag.BoolVar(&valuesVar, "values", false, "Output values")
	flag.Parse()

	level, err := log.ParseLevel(logLevelVar)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	log.SetReportCaller(true)
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.QuoteEmptyFields = true
	customFormatter.FullTimestamp = true
	customFormatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
		repopath := strings.Split(f.File, "/")
		function := strings.Replace(f.Function, "go-pkgdl/", "", -1)
		return fmt.Sprintf("%s\t", function), fmt.Sprintf(" %s:%d\t", repopath[len(repopath)-1], f.Line)
	}

	logrus.SetFormatter(customFormatter)

	supportedTypes := [7]string{"debian", "docker", "generic", "maven", "npm", "pypi", "rpm"}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configFolder := "/.lorenygo/pkgDownloader/"
	configPath := usr.HomeDir + configFolder

	log.Info("Checking existence of download folders for:", supportedTypes)
	for i := 0; i < len(supportedTypes); i++ {
		if _, err := os.Stat(configPath + supportedTypes[i] + "Downloads/"); os.IsNotExist(err) {
			log.Info("No config folder found")
			err = os.MkdirAll(configPath+supportedTypes[i]+"Downloads/", 0700)
			helpers.Check(err, true, "Generating "+configPath+" directory")
		} else {
		}
	}
	log.Debug("Done checking existence for:", supportedTypes)
	//TODO clean up downloads dir beforehand

	masterKey := auth.VerifyMasterKey(configPath + "master.key")
	creds := auth.GetDownloadJSON(configPath+"download.json", masterKey)

	if usernameVar == "" {
		usernameVar = creds.Username
	}
	if apikeyVar == "" {
		apikeyVar = creds.Apikey
	}
	if urlVar == "" {
		urlVar = creds.URL
	}

	if (repoVar == "") && resetVar != true && valuesVar != true {
		log.Error("Must specify -repo <Repository>")
		flag.PrintDefaults()
		os.Exit(0)
	}
	if valuesVar == true {
		log.Info("User: ", creds.Username, "\nURL: ", creds.URL, "\nDownload location: ", creds.DlLocation)
		os.Exit(0)
	}

	if resetVar == true {
		creds = auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
		usernameVar = creds.Username
		apikeyVar = creds.Apikey
		urlVar = creds.URL
	}

	if !auth.VerifyAPIKey(urlVar, usernameVar, apikeyVar) {
		if creds.Username == usernameVar && creds.Apikey == apikeyVar && creds.URL == urlVar {
			log.Warn("Looks like there's an issue with your credentials file. Resetting")
			auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
			creds = auth.GetDownloadJSON(configPath+"download.json", masterKey)
			usernameVar = creds.Username
			apikeyVar = creds.Apikey
			urlVar = creds.URL

		} else {
			log.Error("Looks like there's an issue with your custom credentials. Exiting")
			os.Exit(1)
		}
	}

	//update custom
	creds.Username = usernameVar
	creds.Apikey = apikeyVar
	creds.URL = urlVar

	var repotype, extractedURL, pypiRegistryURL, pypiRepoSuffix = checkTypeAndRepoParams(creds, repoVar)
	pkgRepoDlFolder := repotype + "Downloads"

	//case switch for different package types
	workQueue := list.New()
	var extractedURLStripped = strings.TrimSuffix(extractedURL, "/")
	switch repotype {
	case "debian":
		go func() {
			debian.GetDebianHrefs(extractedURL+"pool/", extractedURLStripped, 1, "", workQueue)
		}()

	case "docker":
		fmt.Println("Work in progress, only works against Docker Hub")
		go func() {
			docker.GetDockerImages(creds.URL, creds.Username, creds.Apikey, repoVar, extractedURL, extractedURLStripped, 1, "", workQueue)
		}()

	case "generic":
		fmt.Println("Work in progress")
		go func() {
			generic.GetGenericHrefs(extractedURL, extractedURLStripped, workQueue)
		}()

	case "maven":
		go func() {
			maven.GetMavenHrefs(extractedURL, extractedURLStripped, workQueue)
		}()

	case "npm":
		npm.GetNPMList(configPath, workQueue)

	case "pypi":
		go func() {
			pypi.GetPypiHrefs(pypiRegistryURL+"/"+pypiRepoSuffix+"/", pypiRegistryURL, extractedURLStripped, workQueue)
		}()

	case "rpm":
		go func() {
			log.Info("rpm takes 10 seconds to init, please be patient")
			//buggy. looks like there is a recursive search that screws it up
			rpm.GetRpmHrefs(extractedURL, extractedURLStripped, workQueue)
		}()

	default:
		log.Println("Unsupported package type", repotype, ". We currently support the following:", supportedTypes)
		os.Exit(0)
	}

	//work queue
	var ch = make(chan interface{}, workersVar+1)
	var wg sync.WaitGroup
	for i := 0; i < workersVar; i++ {
		//wg.Add(1)
		go func(i int) {
			for {

				s, ok := <-ch
				if !ok {
					log.Info("Worker being returned to queue?", i)
					wg.Done()
					//return
				}
				log.Debug("worker ", i, " starting job")
				switch repotype {
				case "debian":
					md := s.(debian.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder, repoVar)
					auth.GetRestAPI("PUT", true, creds.URL+"/api/storage/"+repoVar+"-cache"+md.URL+"?properties=deb.component="+md.Component+";deb.architecture="+md.Architecture+";deb.distribution="+md.Distribution, creds.Username, creds.Apikey, "", nil)

				case "docker":
					md := s.(docker.Metadata)
					docker.DlDockerLayers(creds, md, repoVar, i)

				case "maven":
					md := s.(maven.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder, repoVar)

				case "npm":
					md := s.(npm.Metadata)
					npm.GetNPMMetadata(creds, creds.URL+"/api/npm/"+repoVar+"/", md.ID, md.Package, configPath, pkgRepoDlFolder, i)

				case "pypi":
					md := s.(pypi.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder, repoVar)

				case "rpm":
					md := s.(rpm.Metadata)
					standardDownload(creds, md.URL, md.File, configPath, pkgRepoDlFolder, repoVar)
				}
				log.Debug("worker ", i, " finished job")
			}
			//wg.Done()
		}(i)

	}
	for {
		var count0 = 0
		for workQueue.Len() == 0 {
			log.Info(repotype, " work queue is empty, sleeping for 5 seconds...")
			time.Sleep(5 * time.Second)
			count0++
			if count0 > 10 {
				log.Warn("Looks like nothing's getting put into the workqueue. You might want to enable -debug and take a look")
			}
			if workQueue.Len() > 0 {
				count0 = 0
			}
		}
		s := workQueue.Front().Value
		workQueue.Remove(workQueue.Front())
		ch <- s
	}
	close(ch)
	wg.Wait()

}

func standardDownload(creds auth.Creds, dlURL string, file string, configPath string, pkgRepoDlFolder string, repoVar string) {
	_, headStatusCode, _ := auth.GetRestAPI("HEAD", true, creds.URL+"/"+repoVar+"-cache/"+dlURL, creds.Username, creds.Apikey, "", nil)
	if headStatusCode == 200 {
		log.Debug("skipping, got 200 on HEAD request for %s\n", creds.URL+"/"+repoVar+"-cache/"+dlURL)
		return
	}

	log.Info("Downloading", creds.URL+"/"+repoVar+dlURL)
	auth.GetRestAPI("GET", true, creds.URL+"/"+repoVar+dlURL, creds.Username, creds.Apikey, configPath+pkgRepoDlFolder+"/"+file, nil)
	os.Remove(configPath + pkgRepoDlFolder + "/" + file)

}

//Test if remote repository exists and is a remote
func checkTypeAndRepoParams(creds auth.Creds, repoVar string) (string, string, string, string) {
	repoCheckData, repoStatusCode, _ := auth.GetRestAPI("GET", true, creds.URL+"/api/repositories/"+repoVar, creds.Username, creds.Apikey, "", nil)
	if repoStatusCode != 200 {
		log.Error("Repo", repoVar, "does not exist.")
		os.Exit(0)
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(repoCheckData), &result)
	if result["rclass"] != "remote" {
		log.Error(repoVar, "is a", result["rclass"], "repository and not a remote repository.")
		os.Exit(0)
	}
	if result["packageType"].(string) == "pypi" {
		return result["packageType"].(string), result["url"].(string), result["pyPIRegistryUrl"].(string), result["pyPIRepositorySuffix"].(string)
	}
	return result["packageType"].(string), result["url"].(string), "", ""
}
