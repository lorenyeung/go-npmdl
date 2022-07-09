package gems

import (
	"container/list"
	"encoding/json"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

//Metadata struct of gems metadata object
type Metadata struct {
	URL  string
	File string
	Name string
}

func GetGemsHrefs(creds auth.Creds, url string, base string, gemsWorkerQueue *list.List, flags helpers.Flags) {
	GetGems(creds, flags, gemsWorkerQueue, url, base)
}

func GetGems(creds auth.Creds, flags helpers.Flags, gemsWorkerQueue *list.List, url string, base string) {
	randomSearchMap := make(map[string]string)

	//search for gems via looping through permuations of two letters, alpabetised
	for i := 33; i <= 58; i++ {
		for j := 33; j <= 58; j++ {
			gemsSearchStr := string(rune('A'-1+i)) + string(rune('A'-1+j))
			randomSearchMap[gemsSearchStr] = "taken"
			if !flags.RandomVar {
				log.Debug("Ruby ordered search key:", gemsSearchStr)
				gemsSearch(creds, flags, gemsWorkerQueue, url, base, gemsSearchStr)
			}
		}
	}

	//random search of gems
	if flags.RandomVar {
		for key, value := range randomSearchMap {
			log.Debug("Ruby Random result search Key:", key, " Value:", value)
			gemsSearch(creds, flags, gemsWorkerQueue, url, base, key)
		}
	}
}

type gemData struct {
	GemUri  string `json:"gem_uri"`
	GemName string `json:"name"`
}

func gemsSearch(creds auth.Creds, flags helpers.Flags, gemsWorkerQueue *list.List, url string, base string, gemsSearchStr string) {
	//TODO, search query is paginated for more results
	pg := 1
	for {
		data, _, _ := auth.GetRestAPI("GET", false, url+"api/v1/search.json?query="+gemsSearchStr+"&page="+strconv.Itoa(pg), "", "", "", nil, 0)
		if string(data) == "[]" {
			log.Info("no more pages for ", gemsSearchStr, " moving on to next key")
			return
		}
		var gemSearchApiData []gemData
		err := json.Unmarshal(data, &gemSearchApiData)
		if err != nil {
			fmt.Println(err)
		}
		log.Info("Found ", len(gemSearchApiData), " gems on page ", pg)
		for i := range gemSearchApiData {
			log.Info("Found gem:", gemSearchApiData[i].GemUri)
			var GemsMd Metadata
			GemsMd.URL = strings.TrimPrefix(gemSearchApiData[i].GemUri, base)
			GemsMd.File = gemSearchApiData[i].GemName
			if gemsWorkerQueue.Len() > flags.SleepQueueMaxVar {
				log.Debug("Gems worker queue is at ", gemsWorkerQueue.Len(), ", sleeping for ", flags.WorkerSleepVar, " seconds...")
				time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
			}
			gemsWorkerQueue.PushBack(GemsMd)
		}
		pg++
	}
}
