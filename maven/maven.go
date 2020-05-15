package maven

import (
	"container/list"
	"go-pkgdl/helpers"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of Maven metadata object
type Metadata struct {
	URL  string
	File string
}

//GetMavenHrefs parse hrefs for Maven files
func GetMavenHrefs(url string, base string, MavenWorkerQueue *list.List) string {
	resp, err := http.Get(url)
	// this needs to be threaded better..
	helpers.Check(err, false, "HTTP GET error", helpers.Trace())
	defer resp.Body.Close()

	//fmt.Println(resp) //output from HTML download

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
						GetMavenHrefs(url+strip, base, MavenWorkerQueue)
						break
					}
				}
				checkMaven(t, url, base, MavenWorkerQueue)
			}
		}
	}
}

func checkMaven(t html.Token, url string, base string, MavenWorkerQueue *list.List) {
	//need to consider downloading pom.xml too
	if strings.Contains(t.String(), ".jar") || strings.Contains(t.String(), ".pom") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".jar")) || a.Key == "href" && (strings.HasSuffix(a.Val, ".pom")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				log.Info("queuing download", href, a.Val, MavenWorkerQueue.Len())

				//add Maven metadata to queue
				var MavenMd Metadata
				MavenMd.URL = strings.Replace(href, ":", "", -1)
				MavenMd.File = strings.TrimPrefix(a.Val, ":")
				MavenWorkerQueue.PushBack(MavenMd)
				break
			}
		}
	}
}
