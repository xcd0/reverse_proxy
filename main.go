package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	logfile, err := os.OpenFile("./reverse_proxy.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("cannnot open reverse_proxy.log: %v", err.Error())
	}
	defer logfile.Close()
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetOutput(io.MultiWriter(logfile, os.Stdout))

	m()
}

func m() {
	remote, err := url.Parse("http://localhost:8080/git")
	if err != nil {
		panic(err)
	}

	handler := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.URL)
			r.Host = remote.Host
			w.Header().Set("X-Ben", "Rad")
			p.ServeHTTP(w, r)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", handler(proxy))
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}

func rp() {

	var port int
	var dir string
	var home string

	flag.IntVar(&port, "port", 8080, "port number")
	flag.StringVar(&dir, "dir", "", "sub directory")
	flag.StringVar(&home, "home", "~", "home directory")
	flag.Parse()

	if port < 0 || port > 65535 {
		log.Fatalf("Error : need port number. input port : %v\n", port)
	} else if len(dir) < 1 {
		log.Fatalf("Error : need subdirectory name. input dir : %v\n", dir)
	}

	e := echo.New()

	// http.FileServer(http.Dir("."))
	//e.Static("/memo", "./memo") // memoディレクトリのみならこれ
	e.Static("/", ".")
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Debug = true

	url, _ := url.Parse("http://localhost:" + fmt.Sprintf("%d", port) + "/")
	proxy := httputil.NewSingleHostReverseProxy(url)
	reverseProxyRoutePrefix := "/" + dir
	routerGroup := e.Group(reverseProxyRoutePrefix)
	routerGroup.Use(func(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
		return func(context echo.Context) error {
			req := context.Request()
			res := context.Response().Writer
			// Update the headers to allow for SSL redirection
			req.Host = url.Host
			req.URL.Host = url.Host
			req.URL.Scheme = url.Scheme
			//trim reverseProxyRoutePrefix
			path := req.URL.Path
			req.URL.Path = strings.TrimLeft(path, reverseProxyRoutePrefix)
			// ServeHttp is non blocking and uses a go routine under the hood
			proxy.ServeHTTP(res, req)
			return nil
		}
	})

	e.Logger.Fatal(e.Start(":80"))
}
