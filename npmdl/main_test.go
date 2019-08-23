package main

import (
	"bytes"
	"go-npmdl/auth"
	"go-npmdl/metadata"
	"log"
	"os/user"
	"testing"
)

type Creds struct {
	URL        string
	Username   string
	Apikey     string
	DlLocation string
	Repository string
}

func TestVerifyApiKey(t *testing.T) {
	t.Log("Testing good credentials")
	creds := userForTesting()
	goodResult := auth.VerifyAPIKey(creds.URL, creds.Username, creds.Apikey)
	if goodResult != true {
		t.Errorf("error")
	}

	t.Log("Testing bad credentials")
	badResult := auth.VerifyAPIKey(creds.URL, "tester1", creds.Apikey)
	if badResult != false {
		t.Errorf("error")
	}

}

func TestGetNPMMetadata(t *testing.T) {
	t.Log("Testing NPM Metadata")
	creds := userForTesting()
	metadata.GetNPMMetadata(creds, creds.URL+"/api/npm/"+creds.Repository+"/", "49", "005-http-antao", creds.DlLocation)
}

func TestGenerateDownloadJSON(t *testing.T) {
	t.Log("Testing GenerateDownloadJSON")
	var stdin bytes.Buffer
	//creds := userForTesting()
	stdin.Write([]byte("hunter2\n"))

	// result, err := auth.GenerateDownloadJSON(creds.DlLocation, &stdin)
	// assert.NoError(t, err)
	// assert.Equal(t, "hunter2", result)
}

func userForTesting() auth.Creds {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	data := auth.Creds{
		URL:        "http://localhost:8081/artifactory",
		Username:   "admin",
		Apikey:     "password",
		DlLocation: string(usr.HomeDir + "/testing"),
		Repository: "npm-remote",
	}
	return data
}
