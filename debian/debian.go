package debian

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

//GetDebianHrefs parse hrefs for debian files
func GetDebianHrefs(url string, base string, index int, component string, debianWorkerQueue *list.List) string {
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
						GetDebianHrefs(url+a.Val, base, index+1, component, debianWorkerQueue)
						break
					}
				}

				checkDebian(t, url, base, component, debianWorkerQueue)
			}
		}
	}
}

func checkDebian(t html.Token, url string, base string, component string, debianWorkerQueue *list.List) {
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				parts := strings.Split(href, "_")
				arch := strings.TrimSuffix(parts[len(parts)-1], ".deb")
				dist := "xenial" //hardcoding xenial for now as distribution is stored in the packages file, going to be difficult to parse..
				fmt.Println("queuing download", href, component, arch, dist, debianWorkerQueue.Len())

				//add debian metadata to queue
				var debianMd Metadata
				debianMd.Url = href
				debianMd.Component = component
				debianMd.Architecture = arch
				debianMd.Distribution = dist
				debianMd.File = a.Val
				debianWorkerQueue.PushBack(debianMd)
				break
			}
		}
	}
}
