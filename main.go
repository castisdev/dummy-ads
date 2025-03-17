package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/castisdev/dummy-ads/handler"
	"github.com/gorilla/mux"
)

var cfg *handler.Config

func handleOK(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s, Host:[%s], %v", r.Method, r.RequestURI, r.Host, r.Header)
	w.WriteHeader(http.StatusOK)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s, Host:[%s], %v", r.Method, r.RequestURI, r.Host, r.Header)
	u := strings.TrimPrefix(r.RequestURI, "/redirect")
	http.Redirect(w, r, u, http.StatusFound)
}

func main() {
	cfg = &handler.Config{}
	flag.StringVar(&cfg.Addr, "addr", ":5000", "listen address")
	flag.StringVar(&cfg.Dir, "dir", ".", "directory of ad files")
	flag.StringVar(&cfg.UrlPrefix, "url-prefix", "http://localhost:5000", "url prefix")
	flag.StringVar(&cfg.CertFile, "https-cert", "", "certificate file for https")
	flag.StringVar(&cfg.KeyFile, "https-key", "", "private key file for https")
	flag.BoolVar(&cfg.IgnoreMillisec, "ignore-ms", false, "ignore milliseconds in calculating AD list")
	flag.BoolVar(&cfg.UseRedirect, "use-redirect", false, "use redirect")
	version := flag.Bool("version", false, "print version")
	flag.Parse()

	if *version {
		fmt.Println("dummy-ads version 1.0.2")
		os.Exit(0)
	}

	ah := handler.NewAdListHandler(cfg)
	router := mux.NewRouter()
	router.HandleFunc("/adlist", ah.HandleAdList).Methods("GET")
	router.HandleFunc("/error", handleOK)
	router.PathPrefix("/impression/").Handler(http.HandlerFunc(handleOK))
	router.PathPrefix("/tracking/").Handler(http.HandlerFunc(handleOK))
	router.PathPrefix("/redirect/").Handler(http.HandlerFunc(handleRedirect))
	router.HandleFunc("/clickthrough", handleOK)
	fsrv := http.FileServer(http.Dir(cfg.Dir))
	router.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s, Host:[%s], %v", r.Method, r.RequestURI, r.Host, r.Header)
			fsrv.ServeHTTP(w, r)
		})))
	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	if len(cfg.CertFile) > 0 && len(cfg.KeyFile) > 0 {
		if err := srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}
}
