package debian

import (
	"container/list"
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

//Metadata struct of debian metadata object
type Metadata struct {
	Url          string
	Component    string
	Architecture string
	Distribution string
	File         string
}

//GetDebianHrefs parse hrefs for debian files
func GetDebianHrefs(url string, base string, arti string, repo string, configPath string, creds auth.Creds, index int, component string, dlFolder string, workers int, debianWorkerQueue *list.List) string {
	resp, err := http.Get(url)
	// this needs to be threaded better..
	helpers.Check(err, false, "HTTP GET error")
	defer resp.Body.Close()

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
						if index == 1 {
							component = strings.TrimSuffix(a.Val, "/")
						}
						GetDebianHrefs(url+a.Val, base, arti, repo, configPath, creds, index+1, component, dlFolder, workers, debianWorkerQueue)
						break
					}
				}
				checkDebian(t, url, base, arti, repo, configPath, creds, index, component, dlFolder, workers, debianWorkerQueue)
			}
		}
	}
}

func checkDebian(t html.Token, url string, base string, arti string, repo string, configPath string, creds auth.Creds, index int, component string, dlFolder string, workersVar int, debianWorkerQueue *list.List) {
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				parts := strings.Split(href, "_")
				arch := strings.TrimSuffix(parts[len(parts)-1], ".deb")
				dist := "xenial" //hardcoding xenial for now as distibution is stored in the packages file, going to be difficult to parse..
				fmt.Println("queuing ", arti+"/"+repo+href, component, arch, dist, debianWorkerQueue.Len())

				//add debian metadata to queue
				var debianMd Metadata
				debianMd.Url = href
				debianMd.Component = component
				debianMd.Architecture = arch
				debianMd.Distribution = dist
				debianMd.File = a.Val
				debianWorkerQueue.PushBack(debianMd)
				//auth.GetRestAPI("GET", false, arti+"/"+repo+href, creds.Username, creds.Apikey, configPath+dlFolder+"/"+a.Val)
				//auth.GetRestAPI("PUT", false, arti+"/api/storage/"+repo+"-cache"+href+"?properties=deb.component="+component+";deb.architecture="+arch+";deb.distribution="+dist, creds.Username, creds.Apikey, "")
				//os.Remove(configPath + dlFolder + "/" + a.Val)
				break
			}
		}
	}
}
