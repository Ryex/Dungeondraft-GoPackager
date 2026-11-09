package structures

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"sync"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
)

// PackageTags is the set of tags and tag sets of a package.
//
// Every method locks internally, so the tags can be read from the interface
// while a background task (the tag generation) writes them, and while the
// save timer marshals them. The *PackageTags pointer is stable, but the maps
// inside it are replaced under the lock when the tags are loaded again, so
// callers must not read the Tags/Sets maps directly; use the methods.
type PackageTags struct {
	mu sync.RWMutex

	// map of tag names to array of relative resource paths
	Tags map[string]*Set[string] `json:"tags"`
	// map of set names to array of set names
	Sets map[string]*Set[string] `json:"sets"`
}

func (pt *PackageTags) ListTags() []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return utils.MapKeys(pt.Tags)
}

func (pt *PackageTags) ListSets() []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return utils.MapKeys(pt.Sets)
}

func NewPackageTags() *PackageTags {
	t := &PackageTags{}
	t.Tags = make(map[string]*Set[string])
	t.Sets = make(map[string]*Set[string])
	return t
}

func (pt *PackageTags) TagExists(tag string) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	_, ok := pt.Tags[tag]
	return ok
}

func (pt *PackageTags) SetExists(set string) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	_, ok := pt.Sets[set]
	return ok
}

// Tag tags a set of resources with a tag
func (pt *PackageTags) Tag(tag string, resources ...string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	s, ok := pt.Tags[tag]
	if !ok {
		s = NewSet[string]()
		pt.Tags[tag] = s
	}
	relPaths := slices.Collect(utils.Map(slices.Values(resources), utils.CleanRelativeResourcePath))
	s.AddM(relPaths...)
}

func (pt *PackageTags) AllTags() []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return utils.MapKeys(pt.Tags)
}

func (pt *PackageTags) AllSets() []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return utils.MapKeys(pt.Sets)
}

func (pt *PackageTags) Untag(tag string, resources ...string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	s, ok := pt.Tags[tag]
	if ok {
		relPaths := slices.Collect(utils.Map(slices.Values(resources), utils.CleanRelativeResourcePath))
		s.RemoveM(relPaths...)
	}
}

func (pt *PackageTags) Retag(resource string, tags ...string) {
	relPath := utils.CleanRelativeResourcePath(resource)
	ts := SetFrom(tags)

	pt.mu.Lock()
	defer pt.mu.Unlock()
	for tag := range pt.Tags {
		if ts.Has(tag) {
			pt.Tags[tag].Add(relPath)
		} else {
			pt.Tags[tag].Remove(relPath)
		}
	}
}

func (pt *PackageTags) AddTag(tag string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	_, ok := pt.Tags[tag]
	if !ok {
		pt.Tags[tag] = NewSet[string]()
	}
}

func (pt *PackageTags) AddSet(set string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	_, ok := pt.Sets[set]
	if !ok {
		pt.Sets[set] = NewSet[string]()
	}
}

func (pt *PackageTags) AddTagToSet(set string, tags ...string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
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
	pt.mu.Lock()
	defer pt.mu.Unlock()
	s, ok := pt.Sets[set]
	if ok {
		for _, tag := range tags {
			s.Remove(tag)
		}
	}
}

func (pt *PackageTags) ResourcesFor(tag string) []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	s := pt.Tags[tag]
	if s == nil {
		return nil
	}
	return s.AsSlice()
}

// Set returns a copy of the tags of the given set, so that the caller does
// not read the live set while a background task writes it.
func (pt *PackageTags) Set(set string) *Set[string] {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	s, ok := pt.Sets[set]
	if !ok || s == nil {
		return nil
	}
	return SetFrom(s.AsSlice())
}

func (pt *PackageTags) DeleteTag(tag string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	delete(pt.Tags, tag)
}

func (pt *PackageTags) DeleteSet(set string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	delete(pt.Sets, set)
}

func (pt *PackageTags) TagsFor(resources ...string) *Set[string] {
	relPaths := slices.Collect(utils.Map(slices.Values(resources), utils.CleanRelativeResourcePath))

	pt.mu.RLock()
	defer pt.mu.RUnlock()
	res := NewSet[string]()
	for i, resource := range relPaths {
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
	relPaths := slices.Collect(utils.Map(slices.Values(resources), utils.CleanRelativeResourcePath))

	pt.mu.Lock()
	defer pt.mu.Unlock()
	for tag := range pt.Tags {
		pt.Tags[tag].RemoveM(relPaths...)
	}
}

// Reset clears the tags in place, so that callers that hold the pointer
// keep working.
func (pt *PackageTags) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.Tags == nil {
		pt.Tags = make(map[string]*Set[string])
	}
	if pt.Sets == nil {
		pt.Sets = make(map[string]*Set[string])
	}
	clear(pt.Tags)
	clear(pt.Sets)
}

func (pt *PackageTags) UnmarshalJSON(data []byte) error {
	var parsed struct {
		Tags map[string]*Set[string] `json:"tags"`
		Sets map[string]*Set[string] `json:"sets"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.Tags = parsed.Tags
	pt.Sets = parsed.Sets
	if pt.Tags == nil {
		pt.Tags = make(map[string]*Set[string])
	}
	if pt.Sets == nil {
		pt.Sets = make(map[string]*Set[string])
	}
	return nil
}

func (pt *PackageTags) MarshalJSON() ([]byte, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var buf bytes.Buffer

	buf.WriteString("{")

	buf.WriteString(`"tags":{`)
	tagsKeys := slices.Sorted(maps.Keys(pt.Tags))
	for i, key := range tagsKeys {
		if i != 0 {
			buf.WriteString(",")
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteString(":")
		valJSON, err := json.Marshal(pt.Tags[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}
	buf.WriteString("}")

	buf.WriteString(",")

	buf.WriteString(`"sets":{`)

	setsKeys := slices.Sorted(maps.Keys(pt.Sets))
	for i, key := range setsKeys {
		if i != 0 {
			buf.WriteString(",")
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteString(":")
		valJSON, err := json.Marshal(pt.Sets[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}
	buf.WriteString("}")

	buf.WriteString("}")

	return buf.Bytes(), nil
}
