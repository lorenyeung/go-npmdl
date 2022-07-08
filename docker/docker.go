package docker

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"

	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

type dockerTagMetadata struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type dockerManifestMetadata struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
	} `json:"config"`
	FsLayers []struct {
		BlobSum string `json:"digest"`
	} `json:"layers"`
}

//Metadata docker metadata
type Metadata struct {
	ManifestURLAPI  string
	ManifestURLFile string
	Image           string
	Tag             string
}

//GetDockerImages Docker Engine API search
func GetDockerImages(artURL string, artUser string, artApikey string, dockerRepo string, url string, base string, index int, component string, dockerWorkerQueue *list.List, flags helpers.Flags) string {

	//search upstream only

	//search internet
	//https://github.com/moby/moby/blob/master/client/image_search.go#L17
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error("Docker CLI init error")
		panic(err)
	}
	imageSearch := types.ImageSearchOptions{
		//RegistryAuth:  "http://docker.registry.com",
		//PrivilegeFunc: RequestPrivilegeFunc,
		Filters: filters.NewArgs(),
		Limit:   100,
	}
	randomSearchMap := make(map[string]string)

	//search for docker images via looping through permuations of two letters, alpabetised
	for i := 33; i <= 58; i++ {
		for j := 33; j <= 58; j++ {
			dockerSearchStr := string(rune('A'-1+i)) + string(rune('A'-1+j))
			randomSearchMap[dockerSearchStr] = "taken"
			if !flags.RandomVar {
				log.Debug("Docker ordered search key:", dockerSearchStr)
				results, err := cli.ImageSearch(ctx, dockerSearchStr, imageSearch)
				if err != nil {
					log.Error("Docker image search error:", err)
				}
				dockerSearch(dockerSearchStr, results, artURL, artUser, artApikey, dockerRepo, dockerWorkerQueue, flags)
			}
		}
	}

	//random search of docker images
	if flags.RandomVar {
		for key, value := range randomSearchMap {
			log.Debug("Docker Random result search Key:", key, " Value:", value)
			results, _ := cli.ImageSearch(ctx, key, imageSearch)
			dockerSearch(key, results, artURL, artUser, artApikey, dockerRepo, dockerWorkerQueue, flags)
		}
	}

	return ""
}

func dockerSearch(search string, results []registry.SearchResult, artURL string, artUser string, artApikey string, dockerRepo string, dockerWorkerQueue *list.List, flags helpers.Flags) {
	//gets name, then loops through tags

	for x := range results {
		var tags dockerTagMetadata
		// can probably hit artifactory harder with this call
		data, _, _ := auth.GetRestAPI("GET", true, artURL+"/api/docker/"+dockerRepo+"/v2/"+results[x].Name+"/tags/list", artUser, artApikey, "", nil, 1)

		err := json.Unmarshal([]byte(data), &tags)
		if err != nil {
			fmt.Println("error:" + err.Error())
		}

		for y := range tags.Tags {
			var dockerMd Metadata
			dockerMd.Image = results[x].Name
			dockerMd.Tag = tags.Tags[y]
			dockerMd.ManifestURLAPI = artURL + "/api/docker/" + dockerRepo + "/v2/" + results[x].Name + "/manifests/" + tags.Tags[y]
			dockerMd.ManifestURLFile = artURL + "/" + dockerRepo + "/" + results[x].Name + "/" + tags.Tags[y] + "/manifest.json"
			log.Trace("Docker Queue pushing into queue:", dockerMd.ManifestURLFile)
			dockerWorkerQueue.PushBack(dockerMd)

			for dockerWorkerQueue.Len() > flags.SleepQueueMaxVar {
				log.Debug("Docker worker queue is at ", dockerWorkerQueue.Len(), ", queue max is set to ", flags.SleepQueueMaxVar, ", sleeping for ", flags.WorkerSleepVar, " seconds...")
				time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
			}
			log.Trace("Queue at:", dockerWorkerQueue.Len(), ", resuming docker worker queue")
		}
	}
}

//DlDockerLayers download docker layers
func DlDockerLayers(creds auth.Creds, md Metadata, repo string, workerNum int, generic bool) {
	m := map[string]string{
		"Accept": "application/vnd.docker.distribution.manifest.v2+json",
	}
	log.Debug("Worker ", workerNum, " Getting manifest for first time:", md.ManifestURLAPI, " headers:", m)
	manifest, _, headers := auth.GetRestAPI("GET", true, md.ManifestURLAPI, creds.Username, creds.Apikey, "", m, 1)

	var manifestData dockerManifestMetadata
	err := json.Unmarshal(manifest, &manifestData)
	if err != nil {
		log.Warn("Worker ", workerNum, " error mapping manifest:", md.Image, ":", md.Tag, " skipping further image download due to:"+err.Error())
		//TODO, delete manifest maybe
		return
	}
	log.Trace("Worker ", workerNum, " Manifest data:", string(manifest), md.Image, md.Tag)
	log.Debug("Worker ", workerNum, " Manifest recieved data:", headers, manifestData.Config.Digest, manifestData.Config.MediaType, manifestData.SchemaVersion)
	if manifestData.SchemaVersion != 2 {
		log.Warn("Worker ", workerNum, " encountered schema version ", manifestData.SchemaVersion, " for manifest ", md.Image+":"+md.Tag, " skipping download")
		return
	}
	log.Debug("Worker ", workerNum, " Getting manifest via metadata:", md.ManifestURLAPI)
	auth.GetRestAPI("GET", true, creds.URL+"/api/docker/"+repo+"/v2/"+md.Image+"/manifests/"+md.Tag, creds.Username, creds.Apikey, "", nil, 1)

	//iterate through layer download - tried to do concurrent downloads but this usually rekts Artifactory
	log.Info("Worker ", workerNum, " Got manifest for image ", md.Image, ":", md.Tag, " contains ", len(manifestData.FsLayers), " layers")
	skippedLayers := 0
	for x := range manifestData.FsLayers {
		if x%7 == 0 && x != 0 {
			log.Info("Worker ", workerNum, " Processed ", x, " layers of image ", md.Image, ":", md.Tag)
		}
		headLoc := creds.URL + "/" + repo + "-cache/" + md.Image + "/" + md.Tag + "/" + strings.Replace(manifestData.FsLayers[x].BlobSum, ":", "__", -1)
		log.Debug("Worker ", workerNum, " Getting blob:", manifestData.FsLayers[x].BlobSum)
		_, headStatusCode, _ := auth.GetRestAPI("HEAD", true, headLoc, creds.Username, creds.Apikey, "", nil, 1)
		if headStatusCode == 200 {
			log.Trace("Worker ", workerNum, " skipping current layer ", x, "/", len(manifestData.FsLayers), " got 200 on HEAD request for ", manifestData.FsLayers[x].BlobSum)
			skippedLayers++
			continue
		}
		log.Debug("Worker ", workerNum, " Downloading blob:", manifestData.FsLayers[x].BlobSum)
		blobDownload := ""
		if generic {
			blobDownload = creds.URL + "/" + repo + "/" + md.Image + "/" + md.Tag + "/" + strings.Replace(manifestData.FsLayers[x].BlobSum, ":", "__", -1)
			sha256, _, _ := auth.GetRestAPI("GET", true, creds.URL+"/"+repo+"-cache/"+md.Image+"/"+md.Tag+"/manifest.json.sha256", creds.Username, creds.Apikey, "", nil, 1)
			auth.GetRestAPI("PUT", true, creds.URL+"/api/storage/"+repo+"-cache/"+md.Image+"/"+md.Tag+"/manifest.json?properties=docker.manifest.digest=sha256:"+string(sha256)+";sha256="+string(sha256), creds.Username, creds.Apikey, "", nil, 1)
		} else {
			blobDownload = creds.URL + "/api/docker/" + repo + "/v2/" + md.Image + "/blobs/" + manifestData.FsLayers[x].BlobSum
		}
		auth.GetRestAPI("GET", true, blobDownload, creds.Username, creds.Apikey, "", nil, 1)
		if generic {
			auth.GetRestAPI("PUT", true, creds.URL+"/api/storage/"+repo+"-cache/"+md.Image+"/"+md.Tag+"/"+strings.Replace(manifestData.FsLayers[x].BlobSum, ":", "__", -1)+"?properties=sha256="+strings.Replace(manifestData.FsLayers[x].BlobSum, "sha256:", "", -1), creds.Username, creds.Apikey, "", nil, 1)
			auth.GetRestAPI("GET", true, creds.URL+"/api/docker/"+repo+"/v2/"+md.Image+"/blobs/"+manifestData.FsLayers[x].BlobSum, creds.Username, creds.Apikey, "", nil, 1)

		}
		//TODO maybe some error code if the layers aren't fetching
		log.Debug("Worker ", workerNum, " Finished Getting blob:", manifestData.FsLayers[x].BlobSum)
	}
	log.Info("Worker ", workerNum, " Finished downloading image:", md.Image, ":", md.Tag, ", skipped ", skippedLayers, "/", len(manifestData.FsLayers), " layers as they already existed")
}
