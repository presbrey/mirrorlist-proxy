package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var (
	acsv = flag.String("arch", "x86_64", "")
	rcsv = flag.String("release", "7,7.0.1406", "")

	addr     = flag.String("http", ":80", "")
	next     = flag.String("next", "http://localhost/pub/linux/centos/", "")
	upstream = flag.String("upstream", "http://mirrorlist.centos.org/", "")

	client = &http.Client{
		Timeout: 5 * time.Second,
	}
	archs    = map[string]bool{}
	releases = map[string]bool{}
)

func init() {
	flag.Parse()
	for _, elt := range strings.Split(*acsv, ",") {
		archs[elt] = true
	}
	for _, elt := range strings.Split(*rcsv, ",") {
		releases[elt] = true
	}
}

func main() {
	uri, _ := url.Parse(*upstream)
	proxy := httputil.NewSingleHostReverseProxy(uri)
	server := &http.Server{
		Addr:         *addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req.Host = uri.Host
			req.URL.Host = req.Host
			req.URL.Scheme = uri.Scheme
			req.Header.Del("X-Forwarded-For")
			req.RemoteAddr = ""

			release := req.FormValue("release")
			repo := req.FormValue("repo")
			arch := req.FormValue("arch")
			if req.URL.Path != "/" ||
				!releases[release] ||
				!archs[arch] {
				proxy.ServeHTTP(w, req)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%s%s/%s/%s/\n", *next, release, repo, arch)

			resp, err := client.Get(req.URL.String())
			if err != nil {
				log.Println(err)
				return
			}
			body, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			w.Write(body)
		}),
	}
	log.Fatalln(server.ListenAndServe())
}
