package maven

import (
	"container/list"
	"fmt"
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

//GetMavenHrefs parse hrefs for Maven files
func GetMavenHrefs(url string, base string, index int, component string, MavenWorkerQueue *list.List) string {
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
			fmt.Println(t)
			isAnchor := t.Data == "a"
			if isAnchor {

				// recursive look
				for _, a := range t.Attr {
					if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) {
						if index == 1 {
							component = strings.TrimSuffix(a.Val, "/")
						}
						GetMavenHrefs(url+a.Val, base, index+1, component, MavenWorkerQueue)
						break
					}
				}
				checkMaven(t, url, base, component, MavenWorkerQueue)
			}
		}
	}
}

func checkMaven(t html.Token, url string, base string, component string, MavenWorkerQueue *list.List) {
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				parts := strings.Split(href, "_")
				arch := strings.TrimSuffix(parts[len(parts)-1], ".deb")
				dist := "xenial" //hardcoding xenial for now as distribution is stored in the packages file, going to be difficult to parse..
				fmt.Println("queuing download", href, component, arch, dist, MavenWorkerQueue.Len())

				//add Maven metadata to queue
				var MavenMd Metadata
				MavenMd.Url = href
				MavenMd.Component = component
				MavenMd.Architecture = arch
				MavenMd.Distribution = dist
				MavenMd.File = a.Val
				MavenWorkerQueue.PushBack(MavenMd)
				break
			}
		}
	}
}
