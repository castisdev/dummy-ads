package handler

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/abema/go-mp4"
)

type Loader struct {
	idMaker IDMaker
}

func (lo *Loader) load(dir string) ([]*AdFile, error) {
	adFiles := []*AdFile{}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		mp4, err := lo.parseMp4File(f, dir)
		if err != nil {
			continue
		}
		adFiles = append(adFiles, mp4)
		log.Printf("loaded %v", mp4)
	}
	return adFiles, nil
}

func (lo *Loader) parseMp4File(de fs.DirEntry, dir string) (*AdFile, error) {
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
	return &AdFile{lo.idMaker.get(), fi.Name(), w, h, du}, nil
}

type IDMaker struct {
	v int
}

func (i *IDMaker) get() string {
	i.v++
	return strconv.Itoa(i.v)
}
