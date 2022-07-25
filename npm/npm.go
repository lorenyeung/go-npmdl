package npm

import (
	"container/list"
	"encoding/json"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

//Metadata blah
type artifactMetadata struct {
	Versions map[string]distMetadata
}

//DistMetadata blah
type distMetadata struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

//Metadata for worker queue
type Metadata struct {
	Rows []struct {
		ID string `json:"id"`
	} `json:"rows"`
	ID      string
	Package string
}

//GetNPMMetadata blah
func GetNPMMetadata(creds auth.Creds, URL, packageIndex, packageName, configPath string, dlFolder string, workerNum int, flags helpers.Flags) {
	data, _, _ := auth.GetRestAPI("GET", true, URL+packageName, creds.Username, creds.Apikey, "", nil, 1)
	var metadata = artifactMetadata{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		log.Error("Worker ", workerNum, " error:"+err.Error())
	}
	for i, j := range metadata.Versions {

		s := strings.Split(j.Dist.Tarball, "api/npm/"+flags.RepoVar)
		//fmt.Println(len(s), "length of s") //413 error
		if len(s) > 1 && s[1] != "" {
			_, headStatusCode, _ := auth.GetRestAPI("HEAD", true, creds.URL+"/"+flags.RepoVar+"-cache/"+s[1], creds.Username, creds.Apikey, "", nil, 1)
			if headStatusCode == 200 {
				log.Debug("Worker ", workerNum, " skipping, got 200 on HEAD request for ", creds.URL+"/"+flags.RepoVar+"-cache/"+s[1])
				continue
			}
		}
		if !flags.NpmMetadataVar {
			packageDl := packageIndex + "-" + i + ".tgz"
			log.Info("Worker ", workerNum, " Downloading ", s[1])
			auth.GetRestAPI("GET", true, j.Dist.Tarball, creds.Username, creds.Apikey, configPath+dlFolder+"/"+packageDl, nil, 1)
			err2 := os.Remove(configPath + dlFolder + "/" + packageDl)
			helpers.Check(err2, false, "Deleting file", helpers.Trace())
		}
	}
	helpers.Check(err, false, "Reading", helpers.Trace())
	if err != nil {
		return
	}
}

func GetNPMListNew(creds auth.Creds, flags helpers.Flags, npmWorkerQueue *list.List, url string) {
	randomSearchMap := make(map[string]string)

	//search for files via looping through permuations of two letters, alpabetised
	for i := 33; i <= 58; i++ {
		for j := 33; j <= 58; j++ {
			searchStr := string(rune('A'-1+i)) + string(rune('A'-1+j))
			randomSearchMap[searchStr] = "taken"
			if !flags.RandomVar {
				log.Debug("Ordered search key:", searchStr)
				npmSearch(creds, flags, npmWorkerQueue, url, searchStr)
			}
		}
	}

	//random search of files
	if flags.RandomVar {
		for key, value := range randomSearchMap {
			log.Debug("Random result search Key:", key, " Value:", value)
			npmSearch(creds, flags, npmWorkerQueue, url, key)
		}
	}
}

type npmDataObj struct {
	Data []npmDataPkg `json:"objects"`
}

type npmDataPkg struct {
	Package npmData `json:"package"`
}

type npmData struct {
	Name string `json:"name"`
}

func npmSearch(creds auth.Creds, flags helpers.Flags, npmWorkerQueue *list.List, url string, searchStr string) {
	pg := 1
	size := 250
	counter := 0
	for {
		data, _, _ := auth.GetRestAPI("GET", false, url+"-/v1/search?text="+searchStr+"&from="+strconv.Itoa(pg)+"&size="+strconv.Itoa(size), "", "", "", nil, 0)
		var npmSearchApiData npmDataObj
		err := json.Unmarshal(data, &npmSearchApiData)
		if err != nil {
			fmt.Println(err)
		}
		if len(npmSearchApiData.Data) == 0 {
			log.Info("no more pages for ", searchStr, " moving on to next key")
			return
		}
		//log.Info("Found ", len(npmSearchApiData), " gems on page ", pg)
		for i := range npmSearchApiData.Data {
			log.Info("Found NPM:", npmSearchApiData.Data[i].Package.Name)
			var npmMd Metadata
			npmMd.ID = strconv.Itoa(counter)
			counter++
			npmMd.Package = npmSearchApiData.Data[i].Package.Name
			if npmWorkerQueue.Len() > flags.SleepQueueMaxVar {
				log.Debug("NPM worker queue is at ", npmWorkerQueue.Len(), ", sleeping for ", flags.WorkerSleepVar, " seconds...")
				time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
			}
			npmWorkerQueue.PushBack(npmMd)
		}
		pg = pg + size
	}
}

//GetNPMList function to convert raw list into readable text file
func GetNPMList(configPath string, npmWorkQueue *list.List) {
	if _, err := os.Stat(configPath + "all-npm.json"); os.IsNotExist(err) {
		log.Info("No all-npm.json found, creating...")
		auth.GetRestAPI("GET", false, "https://replicate.npmjs.com/_all_docs", "", "", configPath+"all-npm.json", nil, 1)
	}
	var result Metadata
	file, err := os.Open(configPath + "all-npm.json")
	helpers.Check(err, true, "npm JSON read", helpers.Trace())
	byteValue, _ := ioutil.ReadAll(file)
	json.Unmarshal([]byte(byteValue), &result)
	for i, j := range result.Rows {
		t := strconv.Itoa(i)
		result.ID = t
		result.Package = j.ID
		log.Debug("Get NPM list t:", result.ID, " ID:", result.Package)
		npmWorkQueue.PushBack(result)
	}
}
