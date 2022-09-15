package rpm

import (
	"container/list"
	"go-pkgdl/helpers"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/html"
)

//Metadata struct of RPM metadata object
type Metadata struct {
	URL  string
	File string
}

var junk int
var junkUrls = make(map[string]int)

//GetRpmHrefs parse hrefs for RPM files
func GetRpmHrefs(url string, base string, RpmWorkerQueue *list.List, flags helpers.Flags) string {
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
					if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) && a.Val != "/" && !strings.Contains(a.Val, "://") && a.Val != "centos/" {

						log.Debug("for:", url+a.Val)
						if resp.StatusCode == 404 {
							log.Info("stop recursion on non 200 response code for:", url+a.Val)
							break
						}

						GetRpmHrefs(url+a.Val, base, RpmWorkerQueue, flags)
						break
					}
				}
				//try to skip junk urls
				for i := range t.Attr {
					log.Debug("stuff inside html", t.Attr[i])
					if t.Attr[i].Key == "href" {
						if junkUrls[t.Attr[i].Val] < 2 {
							junk = checkRpm(t, url, base, RpmWorkerQueue, flags, junk)
						}
					}
				}

			}
		}
	}
}

func checkRpm(t html.Token, url string, base string, rpmWorkerQueue *list.List, flags helpers.Flags, junk int) int {
	log.Trace("received url token:", t.String())
	if strings.Contains(t.String(), ".rpm") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".rpm")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				log.Info("queuing ", rpmWorkerQueue.Len(), " for download:", href, a.Val)

				//add RPM metadata to queue
				var RpmMd Metadata
				RpmMd.URL = strings.TrimPrefix(href, "/centos")
				RpmMd.File = a.Val
				rpmWorkerQueue.PushBack(RpmMd)

				for rpmWorkerQueue.Len() > flags.SleepQueueMaxVar {
					log.Info("RPM worker queue is at ", rpmWorkerQueue.Len(), ", queue max is set to ", flags.SleepQueueMaxVar, ", sleeping for ", flags.WorkerSleepVar, " seconds...")
					time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
				}
				log.Trace("Queue at:", rpmWorkerQueue.Len(), ", resuming RPM worker queue")
				break
			}
		}
	} else {
		//there are alot of .filez types, don't want to log everything
		if junk%100 == 0 {
			log.Info("found ", junk, "+ files that aren't .rpm, ignoring them")
		}
		junk++
		for i := range t.Attr {
			log.Debug("stuff inside html", t.Attr[i])
			if t.Attr[i].Key == "href" {
				junkUrls[t.Attr[i].Val]++
				log.Debug(t.Attr[i].Val, " val", junkUrls[t.Attr[i].Val])
			}
		}
		log.Debug("ignoring non .rpm URL received:", t.Attr)
		return junk
	}
	return 0
}
