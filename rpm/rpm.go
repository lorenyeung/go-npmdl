package rpm

import (
	"container/list"
	"fmt"
	"go-pkgdl/helpers"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of RPM metadata object
type Metadata struct {
	URL  string
	File string
}

//GetRpmHrefs parse hrefs for RPM files
func GetRpmHrefs(url string, base string, RpmWorkerQueue *list.List) string {
	resp, err := http.Get(url)
	// this needs to be threaded better..
	helpers.Check(err, false, "HTTP GET error")
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
					if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) && a.Val != "/" && !strings.Contains(a.Val, "://") && a.Val != "centos/" {

						log.Debug("for", url+a.Val)

						GetRpmHrefs(url+a.Val, base, RpmWorkerQueue)
						break
					}
				}
				checkRpm(t, url, base, RpmWorkerQueue)
			}
		}
	}
}

func checkRpm(t html.Token, url string, base string, RpmWorkerQueue *list.List) {

	if strings.Contains(t.String(), ".rpm") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".rpm")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				fmt.Println("queuing download", href, a.Val, RpmWorkerQueue.Len())

				//add RPM metadata to queue
				var RpmMd Metadata
				RpmMd.URL = strings.TrimPrefix(href, "/centos")
				RpmMd.File = a.Val
				RpmWorkerQueue.PushBack(RpmMd)
				break
			}
		}
	}
}
