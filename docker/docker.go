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
func GetDockerImages(artURL string, artUser string, artApikey string, dockerRepo string, url string, base string, index int, component string, dockerWorkerQueue *list.List) string {

	//https://github.com/moby/moby/blob/master/client/image_search.go#L17
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	imageSearch := types.ImageSearchOptions{
		//RegistryAuth:  "http://dlcker registty.com",
		//PrivilegeFunc: RequestPrivilegeFunc,
		Filters: filters.NewArgs(),
		Limit:   100,
	}

	for i := 33; i <= 58; i++ {
		for j := 33; j <= 58; j++ {
			dockerSearchStr := string(rune('A'-1+i)) + string(rune('A'-1+j))
			log.Debug("Docker searching string:", dockerSearchStr)
			results, _ := cli.ImageSearch(ctx, dockerSearchStr, imageSearch)
			dockerSearch(dockerSearchStr, results, artURL, artUser, artApikey, dockerRepo, dockerWorkerQueue)
		}
	}
	return ""
}

func dockerSearch(search string, results []registry.SearchResult, artURL string, artUser string, artApikey string, dockerRepo string, dockerWorkerQueue *list.List) {
	//gets name, then loops through tags

	for x := range results {
		var tags dockerTagMetadata
		// can probably hit artifactory harder with this call
		data, _, _ := auth.GetRestAPI("GET", true, artURL+"/api/docker/"+dockerRepo+"/v2/"+results[x].Name+"/tags/list", artUser, artApikey, "", nil)

		err := json.Unmarshal([]byte(data), &tags)
		if err != nil {
			fmt.Println("error:" + err.Error())
		}
		//var wg sync.WaitGroup
		//wg.Add(len(tags.Tags))

		for y := range tags.Tags {
			//go func(y int) {
			//	defer wg.Done()
			var dockerMd Metadata
			dockerMd.Image = results[x].Name
			dockerMd.Tag = tags.Tags[y]
			dockerMd.ManifestURLAPI = artURL + "/api/docker/" + dockerRepo + "/v2/" + results[x].Name + "/manifests/" + tags.Tags[y]
			dockerMd.ManifestURLFile = artURL + "/" + dockerRepo + "/" + results[x].Name + "/" + tags.Tags[y] + "/manifest.json"
			log.Trace("Docker Queue pushing into queue:", dockerMd.ManifestURLFile)
			dockerWorkerQueue.PushBack(dockerMd)
			//	}(y)
			for dockerWorkerQueue.Len() > 25 {
				log.Debug("Docker worker queue is at ", dockerWorkerQueue.Len(), ", sleeping for 5 seconds...")
				time.Sleep(5 * time.Second)
			}
			log.Trace("Queue at:", dockerWorkerQueue.Len(), ", resuming docker worker queue")
		}
		//wg.Wait()
	}
}

//DownloadDockerLayers download docker layers
func DownloadDockerLayers(creds auth.Creds, md Metadata, repo string, workerNum int) {
	log.Info("Worker ", workerNum, " Getting manifest for first time:", md.ManifestURLAPI)
	m := map[string]string{
		"Accept": "application/vnd.docker.distribution.manifest.v2+json",
	}
	log.Debug("Worker ", workerNum, " Getting manifest via metadata:", md.ManifestURLAPI)
	manifest, _, headers := auth.GetRestAPI("GET", true, md.ManifestURLAPI, creds.Username, creds.Apikey, "", m)

	var manifestData dockerManifestMetadata
	err := json.Unmarshal(manifest, &manifestData)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	log.Trace("Worker ", workerNum, " Manifest headers:", headers, string(manifest), md.Image, md.Tag)
	log.Debug("Worker ", workerNum, " Manifest recieved data:", headers, manifestData.Config.Digest, manifestData.Config.MediaType)
	auth.GetRestAPI("GET", true, creds.URL+"/api/docker/"+repo+"/v2/"+md.Image+"/manifests/"+md.Tag, creds.Username, creds.Apikey, "", nil)

	//iterate through layer download - tried to do concurrent downloads but this usually rekts Artifactory
	//var wg sync.WaitGroup
	//wg.Add(len(manifestData.FsLayers))
	log.Debug("Worker ", workerNum, " Layer count for image ", md.Image, ":", len(manifestData.FsLayers))
	for x := range manifestData.FsLayers {
		//go func(x int) {
		//defer wg.Done()
		headLoc := creds.URL + "/" + repo + "-cache/" + md.Image + "/" + md.Tag + "/" + strings.Replace(manifestData.FsLayers[x].BlobSum, ":", "__", -1)
		log.Debug("Worker ", workerNum, " Getting blob:", manifestData.FsLayers[x].BlobSum)
		_, headStatusCode, _ := auth.GetRestAPI("HEAD", true, headLoc, creds.Username, creds.Apikey, "", nil)
		if headStatusCode == 200 {
			log.Debug("Worker ", workerNum, " skipping, got 200 on HEAD request for ", manifestData.FsLayers[x].BlobSum)
			return
		}
		log.Debug("Worker ", workerNum, " Downloading blob:", manifestData.FsLayers[x].BlobSum)
		blobDownload := creds.URL + "/api/docker/" + repo + "/v2/" + md.Image + "/blobs/" + manifestData.FsLayers[x].BlobSum
		auth.GetRestAPI("GET", true, blobDownload, creds.Username, creds.Apikey, "", nil)
		log.Debug("Worker ", workerNum, " Finished Getting blob:", manifestData.FsLayers[x].BlobSum)
		//}(x)
	}
	log.Info("Worker ", workerNum, " Finished downloading image:", md.Image, ":", md.Tag)
	//wg.Wait()
}
