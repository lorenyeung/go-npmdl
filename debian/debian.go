package debian

import (
	"fmt"
	"go-pkgdl/auth"
	"go-pkgdl/helpers"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

//GetDebianHrefs parse hrefs for debian files
func GetDebianHrefs(url string, base string, arti string, configPath string, creds auth.Creds) string {
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
					if a.Key == "href" && (strings.HasSuffix(a.Val, "/")) {
						GetDebianHrefs(url+a.Val, base, arti, configPath, creds)
						break
					}
				}
				checkDebian(t, url, base, arti, configPath, creds)
			}
		}
	}
}

func checkDebian(t html.Token, url string, base string, arti string, configPath string, creds auth.Creds) {
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)
				go func() {
					fmt.Println("Downloading ", arti+href)
					auth.GetRestAPI("GET", true, arti+href, creds.Username, creds.Apikey, configPath+"debianDownloads/"+a.Val)
					os.Remove(configPath + "debianDownloads/" + a.Val)
				}()
				break
			}
		}
	}
}
