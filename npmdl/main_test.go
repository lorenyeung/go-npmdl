package main

import (
	"go-npmdl/auth"
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

func TestGenerateDownloadJSON(t *testing.T) {

}

func userForTesting() Creds {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	data := Creds{
		URL:        "http://localhost:8081/artifactory",
		Username:   "tester",
		Apikey:     "password",
		DlLocation: string(usr.HomeDir + "/testing"),
		Repository: "testing",
	}
	return data
}
