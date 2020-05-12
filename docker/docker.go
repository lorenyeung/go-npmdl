package docker

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"go-pkgdl/auth"

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
	FsLayers []struct {
		BlobSum string `json:"blobSum"`
	} `json:"fsLayers"`
}

//Metadata docker metadata
type Metadata struct {
	ManifestURLAPI  string
	ManifestURLFile string
	Image           string
	Tag             string
}

//GetDockerImages Docker Engine API search
func GetDockerImages(artURL string, artUser string, artApikey string, dockerRepo string, url string, base string, index int, component string, dockerWorkerQueue *list.List, debug bool) string {

	//https://github.com/moby/moby/blob/master/client/image_search.go#L17

	//https://forums.docker.com/t/registry-v2-catalog/45368/3 trying to get bearer token to get tags, but it doesn't seem to work...

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
			fmt.Println(dockerSearchStr)
			results, _ := cli.ImageSearch(ctx, dockerSearchStr, imageSearch)
			dockerSearch(dockerSearchStr, results, artURL, artUser, artApikey, dockerRepo, dockerWorkerQueue, debug)
		}
	}
	return ""
}

func dockerSearch(search string, results []registry.SearchResult, artURL string, artUser string, artApikey string, dockerRepo string, dockerWorkerQueue *list.List, debug bool) {
	//gets name, then loops through tags
	for x := range results {
		var tags dockerTagMetadata
		// can probably hit artifactory harder with this call
		data, _ := auth.GetRestAPI("GET", true, artURL+"/api/docker/"+dockerRepo+"/v2/"+results[x].Name+"/tags/list", artUser, artApikey, "")

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
			fmt.Println(dockerMd.ManifestURLFile)
			dockerWorkerQueue.PushBack(dockerMd)
			break
		}
	}
}

//DownloadDockerLayers download docker layers
func DownloadDockerLayers(creds auth.Creds, md Metadata, repo string, workerNum int) {
	fmt.Println("Worker", workerNum, "Getting", md.ManifestURLAPI)
	manifest, _ := auth.GetRestAPI("GET", true, md.ManifestURLAPI, creds.Username, creds.Apikey, "")
	var manifestData dockerManifestMetadata
	err := json.Unmarshal(manifest, &manifestData)
	if err != nil {
		fmt.Println("error:" + err.Error())
	}
	//iterate through layer download
	for x := range manifestData.FsLayers {
		fmt.Println(md.Image, md.Tag, manifestData.FsLayers[x].BlobSum)
		blobDownload := creds.URL + "/api/docker/" + repo + "/v2/" + md.Image + "/blobs/" + manifestData.FsLayers[x].BlobSum
		auth.GetRestAPI("GET", true, blobDownload, creds.Username, creds.Apikey, "")
	}
}
