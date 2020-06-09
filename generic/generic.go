package generic

import (
	"container/list"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of Generic object
type Metadata struct {
	URL  string
	File string
}

//GetGenericHrefs parse hrefs for Generic files
func GetGenericHrefs(url string, base string, GenericWorkerQueue *list.List, workerSleepVar int) string {
	if url == "" {
		//must be a local repo, send to generic file generator instead
		for {
			if GenericWorkerQueue.Len() > 10000 {
				log.Debug("Generic worker queue is at ", GenericWorkerQueue.Len(), ", sleeping for ", workerSleepVar, " seconds...")
				time.Sleep(time.Duration(workerSleepVar) * time.Second)
			}
			randomString := RandStringBytesMaskImprSrcSB(10)
			var GenericMd Metadata
			GenericMd.URL = ""
			GenericMd.File = randomString
			GenericWorkerQueue.PushBack(GenericMd)

		}
	} else {
		resp, err := http.Get(url)
		// this needs to be threaded better..
		helpers.Check(err, false, "HTTP GET error", helpers.Trace())
		defer resp.Body.Close()

		log.Debug(resp) //output from HTML download

		z := html.NewTokenizer(resp.Body)
		for {

			tt := z.Next()

			switch {
			case tt == html.ErrorToken:
				// End of the document, we're done
				return ""
			case tt == html.StartTagToken:
				t := z.Token()
				isAnchor := t.Data == "a"
				if isAnchor {

					// recursive look
					for _, a := range t.Attr {
						if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) {

							strip := strings.TrimPrefix(a.Val, ":")
							GetGenericHrefs(url+strip, base, GenericWorkerQueue, workerSleepVar)
							break
						}
					}
					checkGeneric(t, url, base, GenericWorkerQueue)
				}
			}
		}
	}
}

func checkGeneric(t html.Token, url string, base string, GenericWorkerQueue *list.List) {
	//need to consider downloading pom.xml too TODO fix for generic
	if strings.Contains(t.String(), ".jar") || strings.Contains(t.String(), ".pom") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".jar")) || a.Key == "href" && (strings.HasSuffix(a.Val, ".pom")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				fmt.Println("queuing download", href, a.Val, GenericWorkerQueue.Len())

				//add Generic metadata to queue
				var GenericMd Metadata
				GenericMd.URL = strings.Replace(href, ":", "", -1)
				GenericMd.File = strings.TrimPrefix(a.Val, ":")
				GenericWorkerQueue.PushBack(GenericMd)
				break
			}
		}
	}
}

//CreateAndUploadFile generate random string, and upload it to repo
func CreateAndUploadFile(creds auth.Creds, md Metadata, flags helpers.Flags, configPath string, dlFolder string, i int) {
	err := ioutil.WriteFile(configPath+dlFolder+"/"+"file-"+md.File, []byte(md.File), 0644)
	helpers.Check(err, true, "Generating "+md.File+" file", helpers.Trace())
	log.Info("Worker ", i, " Uploading file:", configPath+dlFolder+"/"+"file-"+md.File)

	// headerMap := map[string]string{
	// 	"Content-Type": "text/plain",
	// }

	body, _, _ := auth.GetRestAPI("PUT", true, creds.URL+"/"+flags.RepoVar+"/uploads/"+md.File+"/"+"file-"+md.File, creds.Username, creds.Apikey, configPath+dlFolder+"/"+"file-"+md.File, nil, 0)
	log.Debug("upload returned:", string(body))
	os.Remove(configPath + dlFolder + "/" + "file-" + md.File)
	log.Info("Worker ", i, " Finished Uploading file:", configPath+dlFolder+"/"+"file-"+md.File)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

//RandStringBytesMaskImprSrcSB from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func RandStringBytesMaskImprSrcSB(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
