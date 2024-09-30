package structures

import (
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
)

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

func (pt *PackageTags) AllTags() []string {
	return utils.MapKeys(pt.Tags)
}

func (pt *PackageTags) AllSets() []string {
	return utils.MapKeys(pt.Sets)
}

func (pt *PackageTags) Untag(tag string, resources ...string) {
	s, ok := pt.Tags[tag]
	if ok {
		s.RemoveM(resources...)
	}
}

func (pt *PackageTags) Retag(resource string, tags ...string) {
	ts := SetFrom(tags)
	for tag := range pt.Tags {
		if ts.Has(tag) {
			pt.Tags[tag].Add(resource)
		} else {
			pt.Tags[tag].Remove(resource)
		}
	}
}

func (pt *PackageTags) AddTag(tag string) {
	_, ok := pt.Tags[tag]
	if !ok {
		pt.Tags[tag] = NewSet[string]()
	}
}

func (pt *PackageTags) AddSet(set string) {
	_, ok := pt.Sets[set]
	if !ok {
		pt.Sets[set] = NewSet[string]()
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
	s := pt.Tags[tag]
	return s.AsSlice()
}

func (pt *PackageTags) Set(set string) *Set[string] {
	s := pt.Sets[set]
	return s
}

func (pt *PackageTags) DeleteTag(tag string) {
	delete(pt.Tags, tag)
}

func (pt *PackageTags) DeleteSet(set string) {
	delete(pt.Sets, set)
}

func (pt *PackageTags) TagsFor(resources ...string) *Set[string] {
	res := NewSet[string]()
	for i, resource := range resources {
		cur := NewSet[string]()
		for tag, s := range pt.Tags {
			if s.Has(resource) {
				cur.Add(tag)
			}
		}
		if i == 0 {
			res = cur
		} else {
			res = res.Intersect(cur)
		}
	}
	return res
}

func (pt *PackageTags) ClearTagsFor(resources ...string) {
	for tag := range pt.Tags {
		pt.Tags[tag].RemoveM(resources...)
	}
}
