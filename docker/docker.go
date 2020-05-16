package docker

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"go-pkgdl/auth"

	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
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
func GetDockerImages(artURL string, artUser string, artApikey string, dockerRepo string, url string, base string, index int, component string, dockerWorkerQueue *list.List, random bool, workerSleepVar int) string {

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
			if !random {
				log.Debug("Docker ordered search key:", dockerSearchStr)
				results, _ := cli.ImageSearch(ctx, dockerSearchStr, imageSearch)
				dockerSearch(dockerSearchStr, results, artURL, artUser, artApikey, dockerRepo, dockerWorkerQueue, workerSleepVar)
			}
		}
	}

	//random search of docker images
	if random {
		for key, value := range randomSearchMap {
			log.Debug("Docker Random result search Key:", key, " Value:", value)
			results, _ := cli.ImageSearch(ctx, key, imageSearch)
			dockerSearch(key, results, artURL, artUser, artApikey, dockerRepo, dockerWorkerQueue, workerSleepVar)
		}
	}

	return ""
}

func dockerSearch(search string, results []registry.SearchResult, artURL string, artUser string, artApikey string, dockerRepo string, dockerWorkerQueue *list.List, workerSleepVar int) {
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

			for dockerWorkerQueue.Len() > 75 {
				log.Debug("Docker worker queue is at ", dockerWorkerQueue.Len(), ", sleeping for ", workerSleepVar, " seconds...")
				time.Sleep(time.Duration(workerSleepVar) * time.Second)
			}
			log.Trace("Queue at:", dockerWorkerQueue.Len(), ", resuming docker worker queue")
		}
	}
}

//DlDockerLayers download docker layers
func DlDockerLayers(creds auth.Creds, md Metadata, repo string, workerNum int) {
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
		log.Warn("Worker ", workerNum, " encountered schema version ", manifestData.SchemaVersion, " skipping download")
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
		blobDownload := creds.URL + "/api/docker/" + repo + "/v2/" + md.Image + "/blobs/" + manifestData.FsLayers[x].BlobSum
		auth.GetRestAPI("GET", true, blobDownload, creds.Username, creds.Apikey, "", nil, 1)
		log.Debug("Worker ", workerNum, " Finished Getting blob:", manifestData.FsLayers[x].BlobSum)
	}
	log.Info("Worker ", workerNum, " Finished downloading image:", md.Image, ":", md.Tag, ", skipped ", skippedLayers, "/", len(manifestData.FsLayers), " layers as they already existed")
}
