package generic

import (
	"bytes"
	"container/list"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/docker"
	"go-pkgdl/helpers"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of Generic object
type Metadata struct {
	URL             string
	File            string
	ManifestURLAPI  string
	ManifestURLFile string
	Image           string
	Tag             string
}

//GetGenericHrefs parse hrefs for Generic files
func GetGenericHrefs(url string, base string, GenericWorkerQueue *list.List, genericRepo string, flags helpers.Flags) string {
	if url == "" {
		//must be a local repo, send to generic file generator instead
		for {
			if GenericWorkerQueue.Len() > 10000 {
				log.Debug("Generic worker queue is at ", GenericWorkerQueue.Len(), ", sleeping for ", flags.WorkerSleepVar, " seconds...")
				time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
			}
			randomString := RandStringBytesMaskImprSrcSB(10)
			var GenericMd Metadata
			GenericMd.URL = ""
			GenericMd.File = randomString
			GenericWorkerQueue.PushBack(GenericMd)

		}
	} else {
		needAuth := false
		if flags.UpstreamUsernameVar != "" {
			needAuth = true
		}
		respdata, _, _ := auth.GetRestAPI("GET", needAuth, url, flags.UpstreamUsernameVar, flags.UpstreamApikeyVar, "", nil, 2)
		//resp, err := http.Get(url)
		// this needs to be threaded better..
		//helpers.Check(err, false, "HTTP GET error", helpers.Trace())
		//defer resp.Body.Close()

		resp := ioutil.NopCloser(bytes.NewReader(respdata))

		log.Debug(resp) //output from HTML download

		z := html.NewTokenizer(resp)
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
							GetGenericHrefs(url+strip, base, GenericWorkerQueue, genericRepo, flags)
							break
						}
					}
					checkGeneric(t, url, base, GenericWorkerQueue, genericRepo, flags)
				}
			}
		}
	}
}

func checkGeneric(t html.Token, url string, base string, GenericWorkerQueue *list.List, genericRepo string, flags helpers.Flags) {
	//need to consider downloading pom.xml too TODO fix for generic
	if strings.Contains(t.String(), "manifest.json") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".json")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				log.Info("queuing download ", href, a.Val, GenericWorkerQueue.Len())

				dockerString := strings.Split(href, "/")

				var GenericMd Metadata
				GenericMd.Tag = dockerString[len(dockerString)-2]
				GenericMd.Image = strings.TrimSuffix(href, "/"+GenericMd.Tag+"/manifest.json")
				GenericMd.Image = strings.TrimPrefix(GenericMd.Image, "/")
				//log.Error(dockerString)
				//GenericMd.ManifestURLAPI = flags.URLVar + "/api/docker/" + genericRepo + "/v2/" + GenericMd.Image + "/manifests/" + GenericMd.Tag
				GenericMd.ManifestURLAPI = flags.URLVar + "/" + genericRepo + "/" + GenericMd.Image + "/" + GenericMd.Tag + "/manifest.json"
				GenericMd.ManifestURLFile = flags.URLVar + "/" + genericRepo + "/" + GenericMd.Image + "/" + GenericMd.Tag + "/manifest.json"
				log.Info("Generic Docker Queue pushing into queue:", GenericMd.ManifestURLFile)
				log.Debug("Generic Docker Queue pushing:", GenericMd.Image, " tag:", GenericMd.Tag)
				GenericWorkerQueue.PushBack(GenericMd)

				for GenericWorkerQueue.Len() > 75 {
					log.Debug("Generic worker queue is at ", GenericWorkerQueue.Len(), ", sleeping for ", flags.WorkerSleepVar, " seconds...")
					time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
				}
				log.Trace("Queue at:", GenericWorkerQueue.Len(), ", resuming generic worker queue")
				break
			}
		}

	} else if strings.Contains(t.String(), ".json") {
		log.Debug("layers:", t.String())
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".json")) || a.Key == "href" && (strings.HasPrefix(a.Val, "sha256_")) {
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

func GenericDownload(creds auth.Creds, md Metadata, configPath string, pkgRepoDlFolder string, repoVar string, i int) {

	if md.ManifestURLAPI != "" {
		var dockerMd docker.Metadata
		dockerMd.Image = md.Image
		dockerMd.ManifestURLAPI = md.ManifestURLAPI
		dockerMd.ManifestURLFile = md.ManifestURLFile
		dockerMd.Tag = md.Tag
		docker.DlDockerLayers(creds, dockerMd, repoVar, i, true)
	}

	_, headStatusCode, _ := auth.GetRestAPI("HEAD", true, creds.URL+"/"+repoVar+"-cache/"+md.URL, creds.Username, creds.Apikey, "", nil, 1)
	if headStatusCode == 200 {
		log.Debug("skipping, got 200 on HEAD request for ", creds.URL+"/"+repoVar+"-cache/"+md.URL)
		return
	}

	log.Info("Downloading ", creds.URL+"/"+repoVar+md.URL)
	auth.GetRestAPI("GET", true, creds.URL+"/"+repoVar+md.URL, creds.Username, creds.Apikey, configPath+pkgRepoDlFolder+"/"+md.File, nil, 1)
	os.Remove(configPath + pkgRepoDlFolder + "/" + md.File)
}
