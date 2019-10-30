package debiandl

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func main() {

	url := "http://archive.ubuntu.com/ubuntu"
	getHrefs(url+"/pool/", url, "")

}

func check(e error, panic bool, logs string) {
	if e != nil && panic {
		log.Panicf("%s failed with error:%s\n", logs, e)
	}
	if e != nil && !panic {
		log.Printf("%s failed with error:%s\n", logs, e)
	}
}

func getHrefs(url string, base string, arti string) string {
	resp, err := http.Get(url)
	//bytes, _ := ioutil.ReadAll(resp.Body)

	//fmt.Println("HTML:\n\n", string(bytes))
	if err != nil {
		fmt.Println(err)
	}
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
						//fmt.Println("Found href:", a.Val)
						getHrefs(url+a.Val, base, arti)
						break
					}
				}

				if strings.Contains(t.String(), ".deb") {
					for _, a := range t.Attr {

						// this here can be multi threaded maybe
						if a.Key == "href" && (strings.HasSuffix(a.Val, ".deb")) {
							hrefraw := url + a.Val
							href := strings.TrimPrefix(hrefraw, base)
							go func() {
								fmt.Println("Found href:", arti+href)

								client := http.Client{}
								req, err := http.NewRequest("GET", arti+href, nil)
								//req.SetBasicAuth(userName, apiKey)
								if err != nil {
									fmt.Printf("The HTTP request failed with error %s\n", err)
								} else {
									filepath := "/Users/loreny/go/src/go-debiandl/" + a.Val

									resp, err := client.Do(req)
									check(err, false, "Client check")
									if filepath != "" {
										// Create the file
										out, err := os.Create(filepath)
										check(err, false, "File create")
										defer out.Close()
										_, err = io.Copy(out, resp.Body)
										check(err, true, "File copy")

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
		}
	}
}
