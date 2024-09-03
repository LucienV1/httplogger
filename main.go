package main

import (
        "encoding/json"
        "flag"
        "io"
        "log"
        "net/http"

        "github.com/disgoorg/disgo/webhook"
        agents "github.com/monperrus/crawler-user-agents"
)

var client webhook.Client
var a *bool
var g *bool

func handler(w http.ResponseWriter, r *http.Request) {
        var isbot string
        if !*g {
                if agents.IsCrawler(r.UserAgent()) || r.URL.Path == "/robots.txt" || r.URL.Path == "/favicon.ico" {
                        w.WriteHeader(http.StatusForbidden)
                        return
                }
                isbot = "No"
        } else {
                if agents.IsCrawler(r.UserAgent()) {
                        isbot = "Yes"
                } else {
                        isbot = "No"
                }
        }
		if r.URL.Path == "/robots.txt" || r.URL.Path == "/favicon.ico" {
			return
		}
        if r.Method != "GET" {
                w.WriteHeader(http.StatusMethodNotAllowed)
                return
        } else {
                w.WriteHeader(http.StatusOK)
                w.Write([]byte("<!DOCTYPE html><html><head></head><body><h1>Error 500: Internal server error. Please try agin later.</h1></body></html>"))
                var remote string
                if r.Header.Get("Cf-Connecting-Ip") == "" {
                        if r.Header.Get("X-Forwarded-For") == "" {
                                remote = r.RemoteAddr
                        } else {
                                remote = r.Header.Get("X-Forwarded-For")
                        }
                } else {
                        remote = r.Header.Get("Cf-Connecting-Ip")
                }
                var ct string
                if r.Header.Get("CF-IPCountry") != "" {
                        ct = "\n\t\t\\- Country: " + r.Header.Get("CF-IPCountry")
                }
                body, err := io.ReadAll(r.Body)
                if err != nil {
                        return
                }
                var h string
                if *a {
                        headers, err := json.MarshalIndent(r.Header, "", "  ")
                        if err != nil {
                                return
                        }
                        q := "`"
                        h = `
        \- Headers: 
` + q + q + q + `
` + string(headers) + `
` + q + q + q
                }

                bodyString := string(body)

                client.CreateContent(`HTTP REQUEST:
    	\- Method: ` + r.Method + `
        \- URL: ` + r.URL.String() + `
        \- User-Agent: ` + r.UserAgent() + `
        \- Is Bot: ` + isbot + `
        \- Remote Address: ` + remote + `
        \- Http Version: ` + r.Proto + `
        \- Host: ` + r.Host + `
        \- Referer: ` + r.Referer() + `
        \- Date and Time: ` + r.Header.Get("Date") + ct + `
        \- Lookup IP: https://ip-lookup.net/?` + remote + `
        \- Request Body: ` + bodyString + h,
                )
                return
        }
}

func main() {
        var port string
        var wurl string
        flag.StringVar(&port, "p", "3311", "port to listen on, default is 3311")
        flag.StringVar(&wurl, "w", "", "discord webhook url")
        a = flag.Bool("h", false, "show all headers in json format")
        g = flag.Bool("g", false, "include automated requests")
        flag.Parse()
        var err error
        client, err = webhook.NewWithURL(wurl)
        if err != nil {
                panic(err)
        }
        log.Fatal(http.ListenAndServe(":"+port, http.HandlerFunc(handler)))
}
