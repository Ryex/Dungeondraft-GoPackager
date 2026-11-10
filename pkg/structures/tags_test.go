package structures

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

// TestPackageTagsConcurrentReadWrite exercises the tag maps the way the tag
// generation, the save timer and the interface do at the same time. The maps
// used to be written without a lock, which crashed with "concurrent map
// iteration and map write"; the race detector run of this test pins the fix.
func TestPackageTagsConcurrentReadWrite(t *testing.T) {
	tags := NewPackageTags()

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 100 {
			tags.Tag(fmt.Sprintf("tag%d", i%10), fmt.Sprintf("textures/objects/%d.png", i))
		}
	})
	wg.Go(func() {
		for i := range 100 {
			tags.AddTagToSet("set", fmt.Sprintf("tag%d", i%10))
		}
	})
	wg.Go(func() {
		for range 100 {
			_ = tags.AllTags()
			_ = tags.AllSets()
			_ = tags.TagsFor("textures/objects/1.png")
			_ = tags.Set("set")
			_, _ = json.Marshal(tags)
		}
	})
	wg.Wait()
}
