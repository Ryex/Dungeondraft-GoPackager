package ddpackage

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	log "github.com/sirupsen/logrus"
)

func (p *Package) GenerateTags(generator *GenerateTags) {
	p.generateTags(generator, nil)
}

func (p *Package) GenerateTagsProgress(generator *GenerateTags, progressCallback func(p float64)) {
	p.generateTags(generator, progressCallback)
}

func (p *Package) generateTags(generator *GenerateTags, pcb func(p float64)) {
	for i, fi := range p.fileList {
		if fi.IsTaggable() {
			tagsMap := generator.TagsFromPath(fi.CalcRelPath())
			for tag, sets := range tagsMap {
				p.Tags().Tag(tag, fi.RelPath)
				for _, set := range sets.AsSlice() {
					p.Tags().AddTagToSet(set, tag)
				}
			}
		}
		if pcb != nil {
			pcb(float64(i) / float64(len(p.fileList)))
		}
	}
	p.SaveUnpackedTags()
}

type GenerateTagsOptions struct {
	BuildGlobalTagSet      bool
	GlobalTagSet           string
	BuildTagSetsFromPrefix bool
	PrefixSplitMode        bool
	TagSetPrefrixDelimiter [2]string
	StripTagSetPrefix      bool
	StripExtraPrefix       string
}

type GenerateTags struct {
	options        *GenerateTagsOptions
	tagSetRegex    *regexp.Regexp
	tagSetSplitter func(string) (string, string)
}

func NewGenerateTags(options *GenerateTagsOptions) *GenerateTags {
	gt := &GenerateTags{options: options}
	gt.setupTagSetSplitter()
	return gt
}

// returns a map of tasg to the set of sets they should live in
func (gt *GenerateTags) TagsFromPath(path string) (tagsMap map[string]*structures.Set[string]) {
	tagsMap = make(map[string]*structures.Set[string])
	if gt.tagSetSplitter == nil {
		return
	}

	pathParts := strings.Split(path, "/")

	if len(pathParts) <= 3 {
		// no potential tags in path
		return
	}

	// strip off textures/[objects]/
	pathParts = pathParts[2:]
	// strip off file name
	pathParts = pathParts[:len(pathParts)-1]

	for _, part := range pathParts {
		part = strings.TrimSpace(part)
		first := true
		var sets []string
		var set, rest string
		rest = part
		for first || (set != "") {
			if first {
				first = false
			}
			set, rest = gt.tagSetSplitter(rest)
			if set != "" && !slices.Contains(sets, set) {
				sets = append(sets, set)
			}
			rest = strings.TrimSpace(rest)
		}
		var tag string
		if gt.options.StripTagSetPrefix {
			tag = strings.TrimSpace(rest)
		} else {
			tag = strings.TrimSpace(part)
		}
		if gt.options.StripExtraPrefix != "" {
			tag = strings.TrimPrefix(tag, gt.options.StripExtraPrefix)
		}
		if tag == "" {
			continue
		}
		if _, ok := tagsMap[tag]; !ok {
			tagsMap[tag] = structures.NewSet[string]()
		}
		for _, set := range sets {
			tagsMap[tag].Add(set)
		}
		if gt.options.BuildGlobalTagSet && gt.options.GlobalTagSet != "" {
			tagsMap[tag].Add(gt.options.GlobalTagSet)
		}
	}
	return
}

func (gt *GenerateTags) setupTagSetSplitter() {
	gt.tagSetSplitter = gt.splitNoOp
	if !gt.options.BuildTagSetsFromPrefix {
		return
	}

	if gt.options.TagSetPrefrixDelimiter[0] == "" {
		return
	}
	if gt.options.PrefixSplitMode || gt.options.TagSetPrefrixDelimiter[1] == "" {
		gt.tagSetSplitter = gt.splitSingleSep
		return
	}

	pattern := fmt.Sprintf(
		`^%s(.*?)%s`,
		regexp.QuoteMeta(gt.options.TagSetPrefrixDelimiter[0]),
		regexp.QuoteMeta(gt.options.TagSetPrefrixDelimiter[1]),
	)
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.WithError(err).Warnf("generated Tag separator pattern '%s' is not a valid regex", pattern)
	}
	gt.tagSetRegex = re
	gt.tagSetSplitter = gt.splitStartStopSep
}

func (gt *GenerateTags) splitSingleSep(part string) (set string, rest string) {
	set, rest = utils.SplitOne(part, gt.options.TagSetPrefrixDelimiter[0])
	return
}

func (gt *GenerateTags) splitStartStopSep(part string) (string, string) {
	if gt.tagSetRegex == nil {
		return "", part
	}
	match := gt.tagSetRegex.FindStringSubmatch(part)
	if len(match) == 1 {
		return match[0], strings.Replace(part, match[0], "", 1)
	} else if len(match) == 2 {
		return match[1], strings.Replace(part, match[0], "", 1)
	}
	return "", part
}

func (gt *GenerateTags) splitNoOp(part string) (string, string) {
	return "", part
}
