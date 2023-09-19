package handler

type AdFileSet struct {
	set  []*AdFile
	diff int64
}

type FileSelector struct {
	cfg      *Config
	fileSets []*AdFileSet
}

func (sg *FileSelector) calcSubset(set []*AdFile, subset []*AdFile, index int, diff int64) {
	if len(subset) > 0 {
		sg.fileSets = append(sg.fileSets, &AdFileSet{subset, diff})
	}
	for i := index; i < len(set); i++ {
		newDiff := diff - set[i].duration.Milliseconds()
		if sg.cfg.IgnoreMillisec {
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

func (sg *FileSelector) Select(set []*AdFile, sum int64) []*AdFile {
	sg.fileSets = []*AdFileSet{}
	subsets := sg.subsets(set, sum)
	if len(subsets) == 0 {
		return []*AdFile{}
	}
	best := subsets[0]
	for _, s := range subsets {
		if best.diff > s.diff {
			best = s
		}
	}
	return best.set
}
