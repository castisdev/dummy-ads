package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	mp4 "github.com/abema/go-mp4"
	"github.com/gorilla/mux"
	"github.com/haxqer/vast"
)

var adFiles []*AdFile
var urlPrefix string
var idMaker IDMaker
var selector FileSelector
var ignoreMs bool

type IDMaker struct {
	v int
}

func (i *IDMaker) get() string {
	i.v++
	return strconv.Itoa(i.v)
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

type AdFileSet struct {
	set  []*AdFile
	diff int64
}

type FileSelector struct {
	fileSets []*AdFileSet
}

func (sg *FileSelector) calcSubset(set []*AdFile, subset []*AdFile, index int, diff int64) {
	if len(subset) > 0 {
		sg.fileSets = append(sg.fileSets, &AdFileSet{subset, diff})
	}
	for i := index; i < len(set); i++ {
		newDiff := diff - set[i].duration.Milliseconds()
		if ignoreMs {
			newDiff = diff - int64(set[i].duration.Seconds())*1000
		}
		if newDiff < 0 {
			continue
		}
		subset = append(subset, set[i])
		sg.calcSubset(set, subset, i+1, newDiff)
		subset = subset[:len(subset)-1]
	}
}

func (sg *FileSelector) subsets(set []*AdFile, sum int64) []*AdFileSet {
	subset := []*AdFile{}
	index := 0
	sg.calcSubset(set, subset, index, sum)
	return sg.fileSets
}

func (sg *FileSelector) Select(set []*AdFile, sum int64) *AdFileSet {
	sg.fileSets = []*AdFileSet{}
	subsets := sg.subsets(set, sum)
	if len(subsets) == 0 {
		return nil
	}
	best := subsets[0]
	for _, s := range subsets {
		if best.diff > s.diff {
			best = s
		}
	}
	return best
}

func load(dir string) error {
	adFiles = []*AdFile{}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		mp4, err := parseMp4File(f, dir)
		if err != nil {
			continue
		}
		adFiles = append(adFiles, mp4)
		log.Printf("loaded %v", mp4)
	}
	return nil
}

func parseMp4File(de fs.DirEntry, dir string) (*AdFile, error) {
	if de.IsDir() || path.Ext(de.Name()) != ".mp4" {
		return nil, errors.New("not file")
	}
	fi, err := de.Info()
	if err != nil {
		return nil, err
	}
	abs_dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fpath := path.Join(abs_dir, fi.Name())
	file, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	info, err := mp4.Probe(file)
	if err != nil {
		return nil, err
	}
	w, h := 0, 0
	for _, t := range info.Tracks {
		if t.AVC != nil {
			w, h = int(t.AVC.Width), int(t.AVC.Height)
		}
	}
	if w == 0 || h == 0 {
		return nil, errors.New("not h264")
	}

	du := time.Duration(info.Duration*1000/uint64(info.Timescale)) * time.Millisecond
	return &AdFile{idMaker.get(), fi.Name(), w, h, du}, nil
}

func selectFiles(du time.Duration) []*AdFile {
	s := selector.Select(adFiles, du.Milliseconds())
	if s != nil {
		str := ""
		for _, f := range s.set {
			str += f.String()
		}
		log.Printf("select du[%s]: %s", du, str)
		return s.set
	}
	log.Printf("select du[%s]: not exists", du)
	return []*AdFile{}
}

func URL(elem ...string) string {
	s, _ := url.JoinPath(urlPrefix, elem...)
	return s
}

func vastXml(files []*AdFile) []byte {
	var ads []vast.Ad
	for _, file := range files {
		fileUrl, _ := url.JoinPath(urlPrefix, "files", file.filename)

		ads = append(ads, vast.Ad{
			ID: file.id,
			InLine: &vast.InLine{
				AdSystem: &vast.AdSystem{Name: "dummy-ads"},
				AdTitle:  vast.CDATAString{CDATA: "adTitle"},
				Errors:   []vast.CDATAString{{CDATA: URL("error")}},
				Impressions: []vast.Impression{
					{ID: "11111", URI: URL("impression", "1111")},
					{ID: "11112", URI: URL("impression", "1112")},
				},
				Creatives: []vast.Creative{
					{
						AdID:     file.id,
						Sequence: 1,
						Linear: &vast.Linear{
							Duration: vast.Duration(file.duration),
							TrackingEvents: []vast.Tracking{
								{Event: vast.Event_type_creativeView, URI: URL("tracking", "createview")},
								{Event: vast.Event_type_start, URI: URL("tracking", "start")},
								{Event: vast.Event_type_firstQuartile, URI: URL("tracking", "firstquartile")},
								{Event: vast.Event_type_midpoint, URI: URL("tracking", "midpoint")},
								{Event: vast.Event_type_thirdQuartile, URI: URL("tracking", "thirdquartile")},
								{Event: vast.Event_type_complete, URI: URL("tracking", "complete")},
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
									{URI: URL("clickthrough")},
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

func handleAdList(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query().Get("pod_max_dur")
	du, err := strconv.Atoi(r.URL.Query().Get("pod_max_dur"))
	if err != nil {
		log.Printf("invalid pod_max_dur, %s", qp)
	}
	files := selectFiles(time.Duration(du) * time.Second)
	if len(files) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	bytes := vastXml(files)
	w.Header().Add("Content-Length", strconv.Itoa(len(bytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func handleOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	addr := flag.String("addr", ":5000", "listen address")
	dir := flag.String("dir", ".", "directory of ad files")
	prefix := flag.String("url-prefix", "http://localhost:5000", "url prefix")
	cert := flag.String("https-cert", "", "certificate file for https")
	key := flag.String("https-key", "", "private key file for https")
	ignoreMillisec := flag.Bool("ignore-ms", false, "ignore milliseconds in calculating AD list")
	flag.Parse()

	urlPrefix = *prefix
	ignoreMs = *ignoreMillisec

	if err := load(*dir); err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/adlist", handleAdList).Methods("GET")
	router.HandleFunc("/error", handleOK)
	router.HandleFunc("/impression/{id}", handleOK)
	router.HandleFunc("/tracking/{event}", handleOK)
	router.HandleFunc("/clickthrough", handleOK)
	router.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(*dir))))
	srv := &http.Server{
		Addr:    *addr,
		Handler: router,
	}

	if len(*cert) > 0 && len(*key) > 0 {
		if err := srv.ListenAndServeTLS(*cert, *key); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}
}
