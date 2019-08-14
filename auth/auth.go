package auth

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"files/helpers"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

//Creds struct for creating download.json
type Creds struct {
	URL        string
	Username   string
	Apikey     string
	DlLocation string
	Repository string
}

// VerifyAPIKey for errors
func VerifyAPIKey(urlInput, userName, apiKey string) bool {
	log.Printf("starting VerifyAPIkey request. Testing: %s\n", userName)
	data := GetRestAPI(urlInput+"/api/system/ping", userName, apiKey, "")
	if string(data) == "OK" {
		log.Printf("finished VerifyAPIkey request. Credentials are good to go.")
		return true
	}
	log.Printf("finished VerifyAPIkey request: %s\n", string(data))
	return false
}

// GenerateDownloadJSON (re)generate download JSON. Tested.
func GenerateDownloadJSON(configPath string, regen bool, masterKey string) Creds {
	var creds Creds
	if regen {
		creds = GetDownloadJSON(configPath, masterKey)
	}
	var urlInput, userName, apiKey string
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter your url [%s]: ", creds.URL)
	for {
		urlInput, _ = reader.ReadString('\n')
		urlInput = strings.TrimSuffix(urlInput, "\n")
		if urlInput == "" {
			urlInput = creds.URL
		}
		fmt.Printf("Enter your username [%s]: ", creds.Username)
		userName, _ = reader.ReadString('\n')
		userName = strings.TrimSuffix(userName, "\n")
		if userName == "" {
			userName = creds.Username
		}
		fmt.Print("Enter your API key: ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSuffix(apiKey, "\n")
		if VerifyAPIKey(urlInput, userName, apiKey) {
			break
		} else {
			fmt.Print("Something seems wrong, please try again. Enter your url: ")
		}
	}
	//TODO need to check if directory exists and/or valid directory. trim trailing /
	fmt.Printf("Enter your Download location [%s]: ", creds.DlLocation)
	dlLocationInput, _ := reader.ReadString('\n')
	dlLocationInput = strings.TrimSuffix(dlLocationInput, "\n")
	if dlLocationInput == "" {
		dlLocationInput = creds.DlLocation
	}

	//TODO need to check if repo exists. trim trailing /
	fmt.Printf("Enter your repository [%s]: ", creds.Repository)
	repoInput, _ := reader.ReadString('\n')
	repoInput = strings.TrimSuffix(repoInput, "\n")
	if repoInput == "" {
		repoInput = creds.Repository
	}
	return writeFileDownloadJSON(configPath, urlInput, userName, apiKey, dlLocationInput, repoInput, masterKey)
}

func writeFileDownloadJSON(configPath, urlInput, userName, apiKey, dlLocationInput, repoInput, masterKey string) Creds {
	data := Creds{
		URL:        Encrypt(urlInput, masterKey),
		Username:   Encrypt(userName, masterKey),
		Apikey:     Encrypt(apiKey, masterKey),
		DlLocation: Encrypt(dlLocationInput, masterKey),
		Repository: Encrypt(repoInput, masterKey),
	}
	//should probably encrypt data here
	fileData, err := json.Marshal(data)
	helpers.Check(err, true, "The JSON marshal")
	err2 := ioutil.WriteFile(configPath, fileData, 0600)
	helpers.Check(err2, true, "The JSON write")

	data2 := Creds{
		URL:        urlInput,
		Username:   userName,
		Apikey:     apiKey,
		DlLocation: dlLocationInput,
		Repository: repoInput,
	}

	return data2
}

//GetDownloadJSON get data from DownloadJSON
func GetDownloadJSON(fileLocation string, masterKey string) Creds {
	var result map[string]interface{}
	var resultData Creds
	file, err := os.Open(fileLocation)
	if err != nil {
		log.Print("error:", err)
		resultData = GenerateDownloadJSON(fileLocation, false, masterKey)
	} else {
		//should decrypt here
		defer file.Close()
		byteValue, _ := ioutil.ReadAll(file)
		json.Unmarshal([]byte(byteValue), &result)
		//TODO need to validate some of these fields
		resultData.URL = Decrypt(result["URL"].(string), masterKey)
		resultData.Username = Decrypt(result["Username"].(string), masterKey)
		resultData.Apikey = Decrypt(result["Apikey"].(string), masterKey)
		resultData.DlLocation = Decrypt(result["DlLocation"].(string), masterKey)
		resultData.Repository = Decrypt(result["Repository"].(string), masterKey)
	}
	return resultData
}

//GetRestAPI GET rest APIs response with error handling
func GetRestAPI(urlInput, userName, apiKey, filepath string) []byte {
	client := http.Client{}
	req, err := http.NewRequest("GET", urlInput, nil)
	req.SetBasicAuth(userName, apiKey)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
	} else {

		resp, err := client.Do(req)
		helpers.Check(err, false, "The HTTP response")
		//defer resp.Body.Close()

		if filepath != "" {
			//download percent logger
			sourceSha256 := string(resp.Header["X-Checksum-Sha256"][0])
			fmt.Println(resp.Header["Content-Disposition"][0])
			// Create the file
			out, err := os.Create(filepath)
			helpers.Check(err, false, "File create")
			defer out.Close()

			done := make(chan int64)
			go helpers.PrintDownloadPercent(done, filepath, int64(resp.ContentLength))
			_, err = io.Copy(out, resp.Body)
			helpers.Check(err, true, "The file copy")
			log.Println("Checking downloaded Shasum's match")
			fileSha256 := helpers.ComputeSha256(filepath)
			if sourceSha256 != fileSha256 {
				fmt.Printf("Shasums do not match. Source: %s filesystem %s\n", sourceSha256, fileSha256)
			}
			log.Println("Shasums match.")

		} else {
			data, _ := ioutil.ReadAll(resp.Body)
			return data
		}
	}
	return nil
}

//CreateHash self explanatory
func CreateHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

//Encrypt self explanatory
func Encrypt(dataString string, passphrase string) string {
	data := []byte(dataString)
	block, _ := aes.NewCipher([]byte(CreateHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	helpers.Check(err, true, "Cipher")
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString([]byte(ciphertext))
}

//Decrypt self explanatory
func Decrypt(dataString string, passphrase string) string {
	data, _ := base64.RawURLEncoding.DecodeString(dataString)

	key := []byte(CreateHash(passphrase))
	block, err := aes.NewCipher(key)
	helpers.Check(err, true, "Cipher")
	gcm, err := cipher.NewGCM(block)
	helpers.Check(err, true, "Cipher GCM")
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	helpers.Check(err, true, "GCM open")
	return string(plaintext)
}

//VerifyMasterKey self explanatory
func VerifyMasterKey(configPath string) string {
	_, err := os.Open(configPath)
	var token string
	if err != nil {
		log.Printf("Finding master key failed with error %s\n", err)
		data, err := generateRandomBytes(32)
		helpers.Check(err, true, "Generating new master key")
		err2 := ioutil.WriteFile(configPath, []byte(base64.URLEncoding.EncodeToString(data)), 0600)
		helpers.Check(err2, true, "Master key write")
		log.Println("Successfully generated master key")
		token = base64.URLEncoding.EncodeToString(data)
	} else {
		dat, err := ioutil.ReadFile(configPath)
		helpers.Check(err, true, "Reading master key")
		token = string(dat)
	}
	return token
}

func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}
	return b, nil
}
