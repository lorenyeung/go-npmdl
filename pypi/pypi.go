package pypi

import (
	"container/list"
	"fmt"
	"go-pkgdl/helpers"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/common/log"
	"golang.org/x/net/html"
)

//Metadata struct of PyPi metadata object
type Metadata struct {
	URL  string
	File string
}

//GetPypiHrefs parse PyPi for debian files
func GetPypiHrefs(registry string, registryBase string, url string, flags helpers.Flags, pypiWorkerQueue *list.List) string {
	resp, err := http.Get(registry)
	helpers.Check(err, true, "HTTP GET error", helpers.Trace())
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
						GetPypiHrefs(registryBase+a.Val, registryBase, url, flags, pypiWorkerQueue)
						break
					}
				}
				checkPypi(t, registry, registryBase, url, flags, pypiWorkerQueue)
			}
		}
	}
}

func checkPypi(t html.Token, registry string, registryBase string, url string, flags helpers.Flags, pypiWorkerQueue *list.List) {
	if strings.Contains(t.String(), "#sha256") {
		for _, a := range t.Attr {

			if a.Key == "href" && (strings.Contains(t.String(), "#sha256")) {
				parts := strings.Split(a.Val, "#sha256")
				hrefraw := parts[0]
				href := strings.TrimPrefix(hrefraw, url)
				file := strings.Split(parts[0], "/")

				if pypiWorkerQueue.Len() > flags.SleepQueueMaxVar {
					log.Debug("Docker worker queue is at ", pypiWorkerQueue.Len(), ", sleeping for ", flags.WorkerSleepVar, " seconds...")
					time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
				}

				fmt.Println("Queuing download", href, pypiWorkerQueue.Len())
				//add pypi metadata to queue
				var pypiMd Metadata
				pypiMd.URL = href
				pypiMd.File = file[len(file)-1]
				pypiWorkerQueue.PushBack(pypiMd)
				break
			}
		}
	}
}
