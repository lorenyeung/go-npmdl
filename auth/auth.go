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
	"fmt"
	"go-pkgdl/helpers"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/crypto/ssh/terminal"
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
	log.Debug("starting VerifyAPIkey request. Testing:", userName)
	data, _, _ := GetRestAPI("GET", true, urlInput+"/api/system/ping", userName, apiKey, "", nil)
	if string(data) == "OK" {
		log.Debug("finished VerifyAPIkey request. Credentials are good to go.")
		return true
	}
	log.Debug("finished VerifyAPIkey request:", string(data))
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
	for {
		fmt.Printf("Enter your url [%s]: ", creds.URL)
		urlInput, _ = reader.ReadString('\n')
		urlInput = strings.TrimSuffix(urlInput, "\n")
		if urlInput == "" {
			urlInput = creds.URL
		}
		if !strings.HasPrefix(urlInput, "http") {
			fmt.Println("Please enter a HTTP(s) protocol")
			continue
		}
		if strings.HasSuffix(urlInput, "/") {
			fmt.Println("stripping trailing /")
			urlInput = strings.TrimSuffix(urlInput, "/")
		}
		fmt.Printf("Enter your username [%s]: ", creds.Username)
		userName, _ = reader.ReadString('\n')
		userName = strings.TrimSuffix(userName, "\n")
		if userName == "" {
			userName = creds.Username
		}
		fmt.Print("Enter your API key/Password: ")
		apiKeyByte, _ := terminal.ReadPassword(0)
		apiKey = string(apiKeyByte)
		println()
		if VerifyAPIKey(urlInput, userName, apiKey) {
			break
		} else {
			fmt.Println("Something seems wrong, please try again.")
		}
	}
	dlLocationInput := configPath
	return writeFileDownloadJSON(configPath, urlInput, userName, apiKey, dlLocationInput, masterKey)
}

func writeFileDownloadJSON(configPath, urlInput, userName, apiKey, dlLocationInput, masterKey string) Creds {
	data := Creds{
		URL:        Encrypt(urlInput, masterKey),
		Username:   Encrypt(userName, masterKey),
		Apikey:     Encrypt(apiKey, masterKey),
		DlLocation: Encrypt(dlLocationInput, masterKey),
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
		resultData.URL = Decrypt(result["URL"].(string), masterKey)
		resultData.Username = Decrypt(result["Username"].(string), masterKey)
		resultData.Apikey = Decrypt(result["Apikey"].(string), masterKey)
		resultData.DlLocation = Decrypt(result["DlLocation"].(string), masterKey)
	}
	return resultData
}

//GetRestAPI GET rest APIs response with error handling
func GetRestAPI(method string, auth bool, urlInput, userName, apiKey, filepath string, header map[string]string) ([]byte, int, http.Header) {
	client := http.Client{}
	req, err := http.NewRequest(method, urlInput, nil)
	if auth {
		req.SetBasicAuth(userName, apiKey)
	}
	for x, y := range header {
		log.Debug("Recieved extra header:", x+":"+y)
		req.Header.Set(x, y)
	}
	if err != nil {
		log.Warn("The HTTP request failed with error", err)
	} else {

		resp, err := client.Do(req)
		helpers.Check(err, false, "The HTTP response")

		if err != nil {
			return nil, 0, nil
		}
		if resp.StatusCode != 200 {
			log.Debug("Got status code ", resp.StatusCode, " on ", method, " request for ", urlInput, " continuing")
		}
		//Mostly for HEAD requests
		statusCode := resp.StatusCode
		headers := resp.Header

		if filepath != "" && method == "GET" {
			// Create the file
			out, err := os.Create(filepath)
			helpers.Check(err, false, "File create")
			defer out.Close()

			//done := make(chan int64)
			//go helpers.PrintDownloadPercent(done, filepath, int64(resp.ContentLength))
			_, err = io.Copy(out, resp.Body)
			helpers.Check(err, false, "The file copy")
		} else {
			data, err := ioutil.ReadAll(resp.Body)
			helpers.Check(err, false, "Data read")
			return data, statusCode, headers
		}
	}
	return nil, 0, nil
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
	// TODO if decrypt failure
	//	if err != nil {
	// 	GenerateDownloadJSON(fileLocation, false, passphrase)
	// }
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
		log.Warn("Finding master key failed with error %s\n", err)
		data, err := generateRandomBytes(32)
		helpers.Check(err, true, "Generating new master key")
		err2 := ioutil.WriteFile(configPath, []byte(base64.URLEncoding.EncodeToString(data)), 0600)
		helpers.Check(err2, true, "Master key write")
		log.Info("Successfully generated master key")
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
