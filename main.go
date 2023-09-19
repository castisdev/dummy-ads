package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/castisdev/dummy-ads/handler"
	"github.com/gorilla/mux"
)

func handleOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	cfg := &handler.Config{}
	flag.StringVar(&cfg.Addr, "addr", ":5000", "listen address")
	flag.StringVar(&cfg.Dir, "dir", ".", "directory of ad files")
	flag.StringVar(&cfg.UrlPrefix, "url-prefix", "http://localhost:5000", "url prefix")
	flag.StringVar(&cfg.CertFile, "https-cert", "", "certificate file for https")
	flag.StringVar(&cfg.KeyFile, "https-key", "", "private key file for https")
	flag.BoolVar(&cfg.IgnoreMillisec, "ignore-ms", false, "ignore milliseconds in calculating AD list")
	flag.Parse()

	ah := handler.NewAdListHandler(cfg)
	router := mux.NewRouter()
	router.HandleFunc("/adlist", ah.HandleAdList).Methods("GET")
	router.HandleFunc("/error", handleOK)
	router.HandleFunc("/impression/{id}", handleOK)
	router.HandleFunc("/tracking/{event}", handleOK)
	router.HandleFunc("/clickthrough", handleOK)
	router.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(cfg.Dir))))
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
