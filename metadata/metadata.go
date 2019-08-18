package metadata

import (
	"encoding/json"
	"fmt"
	"go-npmdl/auth"
	"go-npmdl/helpers"
	"log"
	"net/http"
	"os"
	"strings"
)

//Metadata blah
type Metadata struct {
	Versions map[string]DistMetadata
}

//DistMetadata blah
type DistMetadata struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

//GetNPMMetadata blah
func GetNPMMetadata(creds auth.Creds, URL, packageIndex, packageName, configPath string) {
	//log.Printf("Getting metadata for %s%s", URL, packageName)

	//TODO do a head request to skip ahead if it already exists in artifactory
	data := auth.GetRestAPI(true, URL+packageName, creds.Username, creds.Apikey, "")

	var metadata = Metadata{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	for i, j := range metadata.Versions {
		packageDl := packageIndex + "-" + i + ".tgz"

		//TODO can be a problem if you set override base url and the tarball URL no longer matches correctly with creds.URL due to overwrite-base-url header
		s := strings.Split(j.Dist.Tarball, URL)
		res, err := http.Head(creds.URL + "/" + creds.Repository + "-cache/" + s[1])
		//TODO may need to add auth header for HEAD request??
		if err != nil {
			log.Printf("HEAD request got error %s, skipping", err)
			continue
		}
		if res.StatusCode == 200 {
			log.Printf("skipping, got 200 on HEAD request for %s\n", creds.URL+"/"+creds.Repository+"-cache/"+s[1])
			continue
		}
		log.Println(packageIndex, i, j.Dist.Tarball, configPath+"downloads/"+packageDl)
		auth.GetRestAPI(true, j.Dist.Tarball, creds.Username, creds.Apikey, configPath+"downloads/"+packageDl)
		err2 := os.Remove(configPath + "downloads/" + packageDl)
		helpers.Check(err2, false, "Deleting file")
	}
	helpers.Check(err, false, "Reading")
	if err != nil {
		return
	}
}
