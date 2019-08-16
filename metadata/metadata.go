package metadata

import (
	"encoding/json"
	"fmt"
	"go-npmdl/auth"
	"go-npmdl/helpers"
	"log"
	"os"
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
	data := auth.GetRestAPI(true, URL+packageName, creds.Username, creds.Apikey, "")

	var metadata = Metadata{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	for i, j := range metadata.Versions {
		packageDl := packageIndex + "-" + i + ".tgz"
		log.Println(i, j.Dist.Tarball, configPath+"downloads/"+packageDl)
		auth.GetRestAPI(true, j.Dist.Tarball, creds.Username, creds.Apikey, configPath+"downloads/"+packageDl)
		err := os.Remove(configPath + "downloads/" + packageDl)
		helpers.Check(err, false, "Deleting file")
	}
	helpers.Check(err, true, "Reading")
}
