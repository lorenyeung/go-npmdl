package maven

import (
	"container/list"
	"go-pkgdl/helpers"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of Maven metadata object
type Metadata struct {
	URL  string
	File string
}

//GetMavenHrefs parse hrefs for Maven files
func GetMavenHrefs(url string, base string, MavenWorkerQueue *list.List, flags helpers.Flags) string {
	resp, err := http.Get(url)
	// this needs to be threaded better..
	helpers.Check(err, false, "HTTP GET error", helpers.Trace())
	defer resp.Body.Close()

	log.Trace("trace resp", resp) //output from HTML download

	z := html.NewTokenizer(resp.Body)
	for {

		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return ""
		case tt == html.StartTagToken:
			t := z.Token()
			log.Trace("t:", t)
			isAnchor := t.Data == "a"
			if isAnchor {

				// recursive look
				for _, a := range t.Attr {
					if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) {
						strip := strings.TrimPrefix(a.Val, ":")
						log.Debug("strip:", url+strip)
						GetMavenHrefs(url+strip, base, MavenWorkerQueue, flags)
						break
					}
				}
				checkMaven(t, url, base, MavenWorkerQueue, flags)
			}
		}
	}
}

func checkMaven(t html.Token, url string, base string, MavenWorkerQueue *list.List, flags helpers.Flags) {
	//need to consider downloading pom.xml too
	if strings.Contains(t.String(), ".jar") || strings.Contains(t.String(), ".pom") {
		for _, a := range t.Attr {
			//log.Debug(a.Val, a.Key)
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".jar")) || a.Key == "href" && (strings.HasSuffix(a.Val, ".pom")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				if MavenWorkerQueue.Len() > flags.SleepQueueMaxVar {
					log.Debug("Maven worker queue is at ", MavenWorkerQueue.Len(), ", sleeping for ", flags.WorkerSleepVar, " seconds...")
					time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
				}
				log.Info("queuing download", href, a.Val, " queue length:", MavenWorkerQueue.Len())

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
