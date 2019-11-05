package debian

import (
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

//GetDebianHrefs parse hrefs for debian files
func GetDebianHrefs(url string, base string, arti string, repo string, configPath string, creds auth.Creds, index int, component string, dlFolder string, workers int) string {
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
						GetDebianHrefs(url+a.Val, base, arti, repo, configPath, creds, index+1, component, dlFolder, workers)
						break
					}
				}
				checkDebian(t, url, base, arti, repo, configPath, creds, index, component, dlFolder, workers)
			}
		}
	}
}

func checkDebian(t html.Token, url string, base string, arti string, repo string, configPath string, creds auth.Creds, index int, component string, dlFolder string, workersVar int) {
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)

				//var mutex = &sync.Mutex{} //should help with the concurrent map writes issue
				var ch = make(chan string, workersVar+1)
				var wg sync.WaitGroup //multi threading the GET details request
				wg.Add(workersVar)
				for i := 0; i < workersVar; i++ {
					go func(i int) {
						parts := strings.Split(href, "_")
						arch := strings.TrimSuffix(parts[len(parts)-1], ".deb")
						dist := "xenial" //hardcoding xenial for now as distibution is stored in the packages file, going to be difficult to parse..
						fmt.Println("Downloading ", arti+"/"+repo+href, component, arch)
						auth.GetRestAPI("GET", false, arti+"/"+repo+href, creds.Username, creds.Apikey, configPath+dlFolder+"/"+a.Val)
						auth.GetRestAPI("PUT", false, arti+"/api/storage/"+repo+"-cache"+href+"?properties=deb.component="+component+";deb.architecture="+arch+";deb.distribution="+dist, creds.Username, creds.Apikey, "")
						os.Remove(configPath + dlFolder + "/" + a.Val)
					}(i)

				}

				//Now the jobs can be added to the channel, which is used as a queue
				//for scanner.Scan() {
				s := arti + "/" + repo + href
				ch <- s
				//}
				close(ch) // This tells the goroutines there's nothing else to do
				wg.Wait() // Wait for the threads to finish
				break
			}
		}
	}
}
