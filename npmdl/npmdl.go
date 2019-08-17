package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go-npmdl/auth"
	"go-npmdl/helpers"
	"go-npmdl/metadata"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
)

//NpmIds lol
type NpmIds struct {
	Rows []struct {
		ID string `json:"id"`
	} `json:"rows"`
}

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
	if _, err := os.Stat(configPath + "downloads/"); os.IsNotExist(err) {
		log.Println("No config folder found")
		err = os.MkdirAll(configPath+"downloads/", 0700)
		helpers.Check(err, true, "Generating "+configPath+" directory")
	}
	//TODO clean up downloads dir beforehand

	masterKey := auth.VerifyMasterKey(configPath + "master.key")
	creds := auth.GetDownloadJSON(configPath+"download.json", masterKey)
	if !auth.VerifyAPIKey(creds.URL, creds.Username, creds.Apikey) {
		fmt.Println("Looks like there's an issue with your credentials.")
		auth.GenerateDownloadJSON(configPath+"download.json", true, masterKey)
		creds = auth.GetDownloadJSON(configPath+"download.json", masterKey)
	}
	var workers = argsWithoutProg[0]
	if workers == "reload" {
		creds = auth.GetDownloadJSON(configPath+"download.json", masterKey)
	}
	getJSONList(configPath)
	getList(configPath)

	file, err := os.Open(configPath + "all-npm-id.txt")
	helpers.Check(err, true, "npm id read")
	defer file.Close()

	scanner := bufio.NewScanner(file)

	intWorkers, _ := strconv.Atoi(workers)
	//var mutex = &sync.Mutex{} //should help with the concurrent map writes issue
	var ch = make(chan []string, intWorkers+1)
	var wg sync.WaitGroup //multi threading the GET details request
	wg.Add(intWorkers)
	for i := 0; i < intWorkers; i++ {
		go func(i int) {
			for {
				s, ok := <-ch
				if !ok { // if there is nothing to do and the channel has been closed then end the goroutine
					wg.Done()
					return
				}
				metadata.GetNPMMetadata(creds, creds.URL+"/api/npm/"+creds.Repository+"/", s[0], s[1], configPath)
			}
		}(i)
	}

	// Now the jobs can be added to the channel, which is used as a queue
	for scanner.Scan() {
		s := strings.Fields(scanner.Text())
		ch <- s
	}

	close(ch) // This tells the goroutines there's nothing else to do
	wg.Wait() // Wait for the threads to finish

}

func getJSONList(configPath string) {
	if _, err := os.Stat(configPath + "all-npm.json"); os.IsNotExist(err) {
		log.Println("No all-npm.json found")
		auth.GetRestAPI(false, "https://replicate.npmjs.com/_all_docs", "", "", configPath+"all-npm.json")
	}
}

func getList(configPath string) {
	if _, err := os.Stat(configPath + "all-npm-id.txt"); os.IsNotExist(err) {
		log.Println("No all-npm-id.txt found")
		var result NpmIds

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
