package structures

import "github.com/ryex/dungeondraft-gopackager/internal/utils"

type PackageTags struct {
	// map of tag names to array of relative resource paths
	Tags map[string]*Set[string] `json:"tags"`
	// map of set names to array of set names
	Sets map[string]*Set[string] `json:"sets"`
}

func (pt *PackageTags) ListTags() []string {
	return utils.MapKeys(pt.Tags)
}

func (pt *PackageTags) ListSets() []string {
	return utils.MapKeys(pt.Sets)
}

func NewPackageTags() *PackageTags {
	t := &PackageTags{}
	t.Tags = make(map[string]*Set[string])
	t.Sets = make(map[string]*Set[string])
	return t
}

func (pt *PackageTags) TagExists(tag string) bool {
	_, ok := pt.Tags[tag]
	return ok
}

func (pt *PackageTags) SetExists(set string) bool {
	_, ok := pt.Sets[set]
	return ok
}

// tags a set of resources with a tag
func (pt *PackageTags) Tag(tag string, resources ...string) {
	s, ok := pt.Tags[tag]
	if !ok {
		s = NewSet[string]()
		pt.Tags[tag] = s
	}
	s.AddM(resources...)
}

func (pt *PackageTags) Untag(tag string, resources ...string) {
	s, ok := pt.Tags[tag]
	if ok {
		s.RemoveM(resources...)
	}
}

func (pt *PackageTags) AddTagToSet(set string, tags ...string) {
	s, ok := pt.Sets[set]
	if !ok {
		s = NewSet[string]()
		pt.Sets[set] = s
	}
	for _, tag := range tags {
		s.Add(tag)
	}
}

func (pt *PackageTags) RemoveTagFromSet(set string, tags ...string) {
	s, ok := pt.Sets[set]
	if ok {
		for _, tag := range tags {
			s.Remove(tag)
		}
	}
}

func (pt *PackageTags) ResourcesFor(tag string) []string {
	s, _ := pt.Tags[tag]
	return s.AsSlice()
}

func (pt *PackageTags) TagsFor(set string) []string {
	s, _ := pt.Sets[set]
	return s.AsSlice()
}
