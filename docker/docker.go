package docker

import (
	"container/list"
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

//GetDockerImages Docker Engine API search
func GetDockerImages(url string, base string, index int, component string, dockerWorkerQueue *list.List, debug bool) string {

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
	cli.ImageSearch(ctx, "aa", imageSearch)

	return ""
}
