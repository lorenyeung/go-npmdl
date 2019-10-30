package debiandl

import (
	"fmt"
	"go-npmdl/helpers"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

//GetDebianHrefs parse hrefs for debian files
func GetDebianHrefs(url string, base string, arti string, configPath string) string {
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
						GetDebianHrefs(url+a.Val, base, arti, configPath)
						break
					}
				}
				checkDebian(t, url, base, arti, configPath)
			}
		}
	}
}

func checkDebian(t html.Token, url string, base string, arti string, configPath string) {
	if strings.Contains(t.String(), ".deb") {
		for _, a := range t.Attr {
			if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
				hrefraw := url + a.Val
				href := strings.TrimPrefix(hrefraw, base)
				go func() {
					fmt.Println("Downloading ", arti+href)

					client := http.Client{}
					req, err := http.NewRequest("GET", arti+href, nil)
					//req.SetBasicAuth(userName, apiKey)
					if err != nil {
						fmt.Printf("The HTTP request failed with error %s\n", err)
					} else {
						filepath := configPath + "debianDownloads/" + a.Val

						resp, err := client.Do(req)
						helpers.Check(err, false, "Client check")
						if filepath != "" {
							// Create the file
							out, err := os.Create(filepath)
							helpers.Check(err, false, "File create")
							defer out.Close()
							_, err = io.Copy(out, resp.Body)
							helpers.Check(err, true, "File copy")

						} else {
							ioutil.ReadAll(resp.Body)
						}
						os.Remove(filepath)
					}
				}()
				break
			}
		}
	}
}
