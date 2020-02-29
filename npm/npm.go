package npm

import (
	"container/list"
	"encoding/json"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
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
func GetNPMMetadata(creds auth.Creds, URL, packageIndex, packageName, configPath string, dlFolder string) {
	data, _ := auth.GetRestAPI("GET", true, URL+packageName, creds.Username, creds.Apikey, "")
	var metadata = artifactMetadata{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	for i, j := range metadata.Versions {
		packageDl := packageIndex + "-" + i + ".tgz"

		//TODO can be a problem if you set override base url and the tarball URL no longer matches correctly with creds.URL due to overwrite-base-url header
		s := strings.Split(j.Dist.Tarball, URL)
		fmt.Println(len(s), "length of s")
		if len(s) > 1 && s[1] != "" {
			_, headStatusCode := auth.GetRestAPI("HEAD", true, creds.URL+"/"+creds.Repository+"-cache/"+s[1], creds.Username, creds.Apikey, "")
			if headStatusCode == 200 {
				log.Printf("skipping, got 200 on HEAD request for %s\n", creds.URL+"/"+creds.Repository+"-cache/"+s[1])
				continue
			}
		}
		log.Println(packageIndex, i, j.Dist.Tarball, configPath+dlFolder+"/"+packageDl)
		auth.GetRestAPI("GET", true, j.Dist.Tarball, creds.Username, creds.Apikey, configPath+dlFolder+"/"+packageDl)
		err2 := os.Remove(configPath + dlFolder + "/" + packageDl)
		helpers.Check(err2, false, "Deleting file")
	}
	helpers.Check(err, false, "Reading")
	if err != nil {
		return
	}
}

//GetNPMList function to convert raw list into readable text file
func GetNPMList(configPath string, npmWorkQueue *list.List) {
	if _, err := os.Stat(configPath + "all-npm.json"); os.IsNotExist(err) {
		log.Println("No all-npm.json found, creating...")
		auth.GetRestAPI("GET", false, "https://replicate.npmjs.com/_all_docs", "", "", configPath+"all-npm.json")
	}
	var result Metadata
	file, err := os.Open(configPath + "all-npm.json")
	helpers.Check(err, true, "npm JSON read")
	byteValue, _ := ioutil.ReadAll(file)
	json.Unmarshal([]byte(byteValue), &result)
	for i, j := range result.Rows {
		t := strconv.Itoa(i)
		result.ID = t
		result.Package = j.ID
		npmWorkQueue.PushBack(result)
	}
}
