package pypi

import (
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

//GetPypiHrefs parse hrefs for debian files
func GetPypiHrefs(url string, base string, arti string, repo string, configPath string, creds auth.Creds, index int, component string, dlFolder string) string {
	resp, err := http.Get(url)
	helpers.Check(err, true, "HTTP GET error")
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
					log.Println("raw A", a)
					if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) {
						if index == 1 {
							component = strings.TrimSuffix(a.Val, "/")
						}
						GetPypiHrefs(url+a.Val, base, arti, repo, configPath, creds, index+1, component, dlFolder)
						//break
					}
				}
				checkPypi(t, url, base, arti, repo, configPath, creds, index, component, dlFolder)
			}
		}
	}
}

func checkPypi(t html.Token, url string, base string, arti string, repo string, configPath string, creds auth.Creds, index int, component string, dlFolder string) {
	log.Println("check pypi", t)
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)
				go func() {
					parts := strings.Split(href, "_")
					arch := strings.TrimSuffix(parts[len(parts)-1], ".deb")
					dist := "xenial" //hardcoding xenial for now as distibution is stored in the packages file, going to be difficult to parse..
					fmt.Println("Downloading ", arti+"/"+repo+href, component, arch)
					auth.GetRestAPI("GET", true, arti+"/"+repo+href, creds.Username, creds.Apikey, configPath+dlFolder+"/"+a.Val)
					auth.GetRestAPI("PUT", true, arti+"/api/storage/"+repo+"-cache"+href+"?properties=deb.component="+component+";deb.architecture="+arch+";deb.distribution="+dist, creds.Username, creds.Apikey, "")
					os.Remove(configPath + dlFolder + "/" + a.Val)
				}()
				break
			}
		}
	}
}
