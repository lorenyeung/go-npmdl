package main

import (
	"fmt"
	"log"
	"npm-downloader/auth"
	"npm-downloader/helpers"
	"os"
	"os/user"
)

func main() {
	if len(os.Args) == 1 {
		log.Println("Please enter number of workers")
		os.Exit(0)
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	argsWithoutProg := os.Args[1:]
	configFolder := "/.lorenygo/npmdownloader/"
	configPath := usr.HomeDir + configFolder
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Println("No config folder found")
		err = os.Mkdir(configPath, 0700)
		helpers.Check(err, true, "Generating "+configPath+" directory")
	}

	masterKey := auth.VerifyMasterKey(configPath + "master.key")
	creds := auth.GetDownloadJSON(configPath+"download.json", masterKey)
	if !auth.VerifyAPIKey(creds.URL, creds.Username, creds.Apikey) {
		fmt.Println("Looks like there's an issue with your credentials.")
		auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
		creds = auth.GetDownloadJSON(configPath+"download.json", masterKey)
	}
	var workers = argsWithoutProg[0]
	log.Printf(configPath, workers)

}

func getJSONList(configPath string) {
	if _, err := os.Stat(configPath + "all-npm.json"); os.IsNotExist(err) {
		log.Println("No all-npm.json found")
		auth.GetRestAPI("https://replicate.npmjs.com/_all_docs", "", "", configPath+"all-npm.json")
	}
}

func getList(configPath string) {
	if _, err := os.Stat(configPath + "all-npm-id.txt"); os.IsNotExist(err) {
		log.Println("No all-npm-id.txt found")
		err = os.Mkdir(configPath, 0700)
		Check(err, true, "Generating "+configPath+" directory")
	}
}
