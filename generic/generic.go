package generic

import (
	"container/list"
	"fmt"
	"go-pkgdl/helpers"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of Generic object
type Metadata struct {
	URL  string
	File string
}

//GetGenericHrefs parse hrefs for Generic files
func GetGenericHrefs(url string, base string, GenericWorkerQueue *list.List) string {
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
						GetGenericHrefs(url+strip, base, GenericWorkerQueue)
						break
					}
				}
				checkGeneric(t, url, base, GenericWorkerQueue)
			}
		}
	}
}

func checkGeneric(t html.Token, url string, base string, GenericWorkerQueue *list.List) {
	//need to consider downloading pom.xml too
	if strings.Contains(t.String(), ".jar") || strings.Contains(t.String(), ".pom") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".jar")) || a.Key == "href" && (strings.HasSuffix(a.Val, ".pom")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				fmt.Println("queuing download", href, a.Val, GenericWorkerQueue.Len())

				//add Maven metadata to queue
				var MavenMd Metadata
				MavenMd.URL = strings.Replace(href, ":", "", -1)
				MavenMd.File = strings.TrimPrefix(a.Val, ":")
				GenericWorkerQueue.PushBack(MavenMd)
				break
			}
		}
	}
}
