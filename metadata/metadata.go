package metadata

import (
	"encoding/json"
	"fmt"
	"go-npmdl/auth"
	"go-npmdl/helpers"
	"log"
)

//Metadata blah
type Metadata struct {
	ID             string   `json:"_id"`
	Rev            string   `json:"_rev"`
	Name           string   `json:"_name"`
	DistTags       struct{} `json:"dist_tags"`
	Versions       map[string]DistMetadata
	Attachments    struct{} `json:"_attachments"`
	Readme         string
	Maintainers    []map[string]string
	Time           struct{} `json:"time"`
	Users          struct{} `json:"users"`
	ReadmeFilename string   `json:"readmeFilename"`
	License        string   `json:"license"`
}

//Maintainers blah
type Maintainers struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

//
type DistMetadata struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

//GetNPMMetadata blah
func GetNPMMetadata(creds auth.Creds, URL, packageName string) {
	log.Printf("Getting metadata for %s%s", URL, packageName)
	data := auth.GetRestAPI(true, URL+packageName, creds.Username, creds.Apikey, "")

	var metadata = Metadata{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	for i, j := range metadata.Versions {
		fmt.Println(i, j.Dist.Tarball)
	}
	helpers.Check(err, true, "Reading")
}
