package npm

import (
	"bufio"
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
type metadata struct {
	Versions map[string]distMetadata
}

//DistMetadata blah
type distMetadata struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

type npmIds struct {
	Rows []struct {
		ID string `json:"id"`
	} `json:"rows"`
}

//GetNPMMetadata blah
func GetNPMMetadata(creds auth.Creds, URL, packageIndex, packageName, configPath string, dlFolder string) {
	data, _ := auth.GetRestAPI("GET", true, URL+packageName, creds.Username, creds.Apikey, "")
	var metadata = metadata{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	for i, j := range metadata.Versions {
		packageDl := packageIndex + "-" + i + ".tgz"

		//TODO can be a problem if you set override base url and the tarball URL no longer matches correctly with creds.URL due to overwrite-base-url header
		s := strings.Split(j.Dist.Tarball, URL)
		if s[1] != "" {
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

//GetNPMJSONList function to collect raw npm list of packages
func GetNPMJSONList(configPath string) {
	if _, err := os.Stat(configPath + "all-npm.json"); os.IsNotExist(err) {
		log.Println("No all-npm.json found, creating...")
		auth.GetRestAPI("GET", false, "https://replicate.npmjs.com/_all_docs", "", "", configPath+"all-npm.json")
	}
}

//GetNPMList function to convert raw list into readable text file
func GetNPMList(configPath string) {
	if _, err := os.Stat(configPath + "all-npm-id.txt"); os.IsNotExist(err) {
		log.Println("No all-npm-id.txt found, creating...")
		var result npmIds

		file, err := os.Open(configPath + "all-npm.json")

		helpers.Check(err, true, "npm JSON read")
		writeFile, err := os.OpenFile(configPath+"all-npm-id.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		helpers.Check(err, true, "npm id write")
		defer file.Close()
		datawriter := bufio.NewWriter(writeFile)
		byteValue, _ := ioutil.ReadAll(file)
		json.Unmarshal([]byte(byteValue), &result)
		for i, j := range result.Rows {
			t := strconv.Itoa(i)
			_, _ = datawriter.WriteString(string(t) + " " + j.ID + "\n")
		}
		datawriter.Flush()
		writeFile.Close()
	}
}
