package handler

import "math/rand"

type AdFileSet struct {
	set  []*AdFile
	diff int64
}

type FileSelector struct {
	cfg         *Config
	minDiffSets []*AdFileSet
}

func CloneAdFiles(files []*AdFile) []*AdFile {
	ret := []*AdFile{}
	for _, f := range files {
		ret = append(ret, f.Clone())
	}
	return ret
}

func (sg *FileSelector) calcSubset(set []*AdFile, subset []*AdFile, index int, diff int64) {
	if len(subset) > 0 {
		if len(sg.minDiffSets) == 0 || sg.minDiffSets[0].diff > diff {
			sg.minDiffSets = []*AdFileSet{{CloneAdFiles(subset), diff}}
		} else if sg.minDiffSets[0].diff == diff {
			sg.minDiffSets = append(sg.minDiffSets, &AdFileSet{CloneAdFiles(subset), diff})
		}
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

func (sg *FileSelector) getMinDiffSets(set []*AdFile, sum int64) []*AdFileSet {
	subset := []*AdFile{}
	index := 0
	sg.calcSubset(set, subset, index, sum)
	return sg.minDiffSets
}

func (sg *FileSelector) Select(set []*AdFile, sum int64) []*AdFile {
	sg.minDiffSets = []*AdFileSet{}
	sets := sg.getMinDiffSets(set, sum)
	if len(sets) == 0 {
		return []*AdFile{}
	}
	return sets[rand.Intn(len(sets))].set
}
