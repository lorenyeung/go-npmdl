package main

import (
	"files/helpers"
	"fmt"
	"log"
	"npm-downloader/auth"
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
