package handler

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/haxqer/vast"
)

type Config struct {
	Addr           string
	Dir            string
	UrlPrefix      string
	CertFile       string
	KeyFile        string
	IgnoreMillisec bool
}

type AdFile struct {
	id       string
	filename string
	width    int
	height   int
	duration time.Duration
}

func (a *AdFile) String() string {
	return fmt.Sprintf("{%s %s %d %d %s}", a.id, a.filename, a.width, a.height, a.duration)
}

type AdListHandler struct {
	cfg     *Config
	adFiles []*AdFile
}

func NewAdListHandler(cfg *Config) *AdListHandler {
	var lo Loader
	files, err := lo.load(cfg.Dir)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		log.Fatalf("not exists mp4 file in %s", cfg.Dir)
	}
	return &AdListHandler{cfg, files}
}

func (ah *AdListHandler) selectFiles(du time.Duration) []*AdFile {
	selector := FileSelector{cfg: ah.cfg}
	s := selector.Select(ah.adFiles, du.Milliseconds())
	str := ""
	for _, f := range s {
		str += f.String()
	}
	log.Printf("select du[%s]: %s", du, str)
	return s
}

func (ah *AdListHandler) URL(elem ...string) string {
	s, _ := url.JoinPath(ah.cfg.UrlPrefix, elem...)
	return s
}

func (ah *AdListHandler) vastXml(files []*AdFile) []byte {
	var ads []vast.Ad
	for _, file := range files {
		fileUrl, _ := url.JoinPath(ah.cfg.UrlPrefix, "files", file.filename)

		ads = append(ads, vast.Ad{
			ID: file.id,
			InLine: &vast.InLine{
				AdSystem: &vast.AdSystem{Name: "dummy-ads"},
				AdTitle:  vast.CDATAString{CDATA: "adTitle"},
				Errors:   []vast.CDATAString{{CDATA: ah.URL("error")}},
				Impressions: []vast.Impression{
					{ID: "11111", URI: ah.URL("impression", "1111")},
					{ID: "11112", URI: ah.URL("impression", "1112")},
				},
				Creatives: []vast.Creative{
					{
						AdID:     file.id,
						Sequence: 1,
						Linear: &vast.Linear{
							Duration: vast.Duration(file.duration),
							TrackingEvents: []vast.Tracking{
								{Event: vast.Event_type_creativeView, URI: ah.URL("tracking", "createview")},
								{Event: vast.Event_type_start, URI: ah.URL("tracking", "start")},
								{Event: vast.Event_type_firstQuartile, URI: ah.URL("tracking", "firstquartile")},
								{Event: vast.Event_type_midpoint, URI: ah.URL("tracking", "midpoint")},
								{Event: vast.Event_type_thirdQuartile, URI: ah.URL("tracking", "thirdquartile")},
								{Event: vast.Event_type_complete, URI: ah.URL("tracking", "complete")},
							},
							MediaFiles: []vast.MediaFile{
								{
									Delivery: "progressive",
									Type:     "video/mp4",
									Width:    file.width,
									Height:   file.height,
									URI:      fileUrl,
								},
							},
							VideoClicks: &vast.VideoClicks{
								ClickThroughs: []vast.VideoClick{
									{URI: ah.URL("clickthrough")},
								},
							},
						},
					},
				},
			},
		})
	}

	v := vast.VAST{
		Version: "3.0",
		Ads:     ads,
	}
	b, _ := xml.Marshal(v)
	return b
}

func (ah *AdListHandler) HandleAdList(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query().Get("pod_max_dur")
	du, err := strconv.Atoi(r.URL.Query().Get("pod_max_dur"))
	if err != nil {
		log.Printf("invalid pod_max_dur, %s", qp)
	}
	files := ah.selectFiles(time.Duration(du) * time.Second)
	if len(files) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	bytes := ah.vastXml(files)
	w.Header().Add("Content-Length", strconv.Itoa(len(bytes)))
	w.Header().Add("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
