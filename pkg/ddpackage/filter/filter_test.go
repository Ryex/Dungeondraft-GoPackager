package filter

import (
	"testing"

	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertGroup(t *testing.T, filter ResourceFilter, groupType ResourceFilterGroupType, filters []ResourceFilter) {
}

func TestParseFilterSingleNameFuzzy(t *testing.T) {
	str := "bird"
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      "bird",
			MatchCase: FilterMatchCaseInsensitive,
			MatchType: FilterMatchFuzzy,
		},
		filter,
	)

	str = "%bird"
	filter, err = ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      "bird",
			MatchCase: FilterMatchCaseSensitive,
			MatchType: FilterMatchFuzzy,
		},
		filter,
	)
}

func TestParseFilterSingleNameExcape(t *testing.T) {
	str := `\^\$bird"\""`
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      `^$bird"`,
			MatchCase: FilterMatchCaseInsensitive,
			MatchType: FilterMatchFuzzy,
		},
		filter)
}

func TestParseFilterSingleNameBegin(t *testing.T) {
	str := "^bir"
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      "bir",
			MatchCase: FilterMatchCaseInsensitive,
			MatchType: FilterMatchBegin,
		},
		filter,
	)

	str = "%^bir"
	filter, err = ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      "bir",
			MatchCase: FilterMatchCaseSensitive,
			MatchType: FilterMatchBegin,
		},
		filter,
	)
}

func TestParseFilterSingleNameEnd(t *testing.T) {
	str := "$fire"
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      "fire",
			MatchCase: FilterMatchCaseInsensitive,
			MatchType: FilterMatchEnd,
		},
		filter,
	)

	str = "$%fire"
	filter, err = ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t,
		&ResourceFilterName{
			Name:      "fire",
			MatchCase: FilterMatchCaseSensitive,
			MatchType: FilterMatchEnd,
		},
		filter,
	)
}

func TestParseFilterMutiNameFuzzy(t *testing.T) {
	str := "bird rock"
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t, filter, &ResourceFilterOr{
		Filters: []ResourceFilter{
			&ResourceFilterName{
				Name:      "bird",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
			&ResourceFilterName{
				Name:      "rock",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
		},
	},
	)
}

func TestParseFilterTagNameFuzzy(t *testing.T) {
	str := "#bird rock"
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t, filter, &ResourceFilterOr{
		Filters: []ResourceFilter{
			&ResourceFilterTag{
				Tag:       "bird",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
			&ResourceFilterName{
				Name:      "rock",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
		},
	},
	)
}

func TestParseFilterGlobNameFuzzy(t *testing.T) {
	str := "/**/bird*.png rock"
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	regxp, _ := structures.GlobToRelPathRegexp("**/bird*.png")
	assert.Equal(t, filter, &ResourceFilterOr{
		Filters: []ResourceFilter{
			&ResourceFilterRelPathGlob{
				RelPathGlobRegexp: regxp,
			},
			&ResourceFilterName{
				Name:      "rock",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
		},
	},
	)
}

func TestParseFilterGroupCombo(t *testing.T) {
	str := `bunny(&#bird rock"()")`
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t, filter, &ResourceFilterOr{
		Filters: []ResourceFilter{
			&ResourceFilterName{
				Name:      "bunny",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
			&ResourceFilterAnd{
				Filters: []ResourceFilter{
					&ResourceFilterTag{
						Tag:       "bird",
						MatchCase: FilterMatchCaseInsensitive,
						MatchType: FilterMatchFuzzy,
					},
					&ResourceFilterName{
						Name:      "rock()",
						MatchCase: FilterMatchCaseInsensitive,
						MatchType: FilterMatchFuzzy,
					},
				},
			},
		},
	},
	)
}

func TestParseFilterGroupCollapse(t *testing.T) {
	str := `(& #bird rock)`
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.Equal(t, &ResourceFilterAnd{
		Filters: []ResourceFilter{
			&ResourceFilterTag{
				Tag:       "bird",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
			&ResourceFilterName{
				Name:      "rock",
				MatchCase: FilterMatchCaseInsensitive,
				MatchType: FilterMatchFuzzy,
			},
		},
	},
		filter,
	)
}

func TestParseFilterNamedFilters(t *testing.T) {
	str := `@IsTexture
	        @IsObject
	        @IsMetadata
	        @IsData
	        @IsWall
	        @IsWallData
	        @IsTileset
	        @IsTilesetData
	        @IsThumbnail
	        @IsCave
	        @IsLight
	        @IsMaterial
	        @IsPath
	        @IsPattern
	        @IsPortal
	        @IsTerrain
	        @IsTaggable`
	filter, err := ParseResourceFilter(str)
	assert.Nil(t, err)
	assert.EqualExportedValues(t, filter, &ResourceFilterOr{
		Filters: []ResourceFilter{
			ResourceFilterIsTexture,
			ResourceFilterIsObject,
			ResourceFilterIsMetadata,
			ResourceFilterIsData,
			ResourceFilterIsWall,
			ResourceFilterIsWallData,
			ResourceFilterIsTileset,
			ResourceFilterIsTilesetData,
			ResourceFilterIsThumbnail,
			ResourceFilterIsCave,
			ResourceFilterIsLight,
			ResourceFilterIsMaterial,
			ResourceFilterIsPath,
			ResourceFilterIsPattern,
			ResourceFilterIsPortal,
			ResourceFilterIsTerrain,
			ResourceFilterIsTaggable,
		},
	},
	)
}

// groupCollapseSamples returns sample resources to compare the matching
// behaviour of two parsed filters on.
func groupCollapseSamples() []*structures.FileInfo {
	return []*structures.FileInfo{
		{ResPath: "res://packs/testpack/textures/objects/Rock.png"},
		{ResPath: "res://packs/testpack/textures/objects/Bedrock.png"},
		{ResPath: "res://packs/testpack/textures/objects/Tree.png"},
		{ResPath: "res://packs/testpack/textures/tilesets/Grass.png"},
		{ResPath: "res://packs/testpack/textures/walls/Stone.png"},
	}
}

// TestParseFilterGroupedSingleNameBehavesLikeBareName checks that the group
// of "(rock)" collapses to its single filter instead of being wrapped in
// another group (filter.go: "filter out empty filters before deciding
// whether the group collapses"), and that it then matches exactly like the
// bare "rock".
func TestParseFilterGroupedSingleNameBehavesLikeBareName(t *testing.T) {
	bare, err := ParseResourceFilter("rock")
	require.Nil(t, err)
	require.Equal(t,
		&ResourceFilterName{
			Name:      "rock",
			MatchCase: FilterMatchCaseInsensitive,
			MatchType: FilterMatchFuzzy,
		},
		bare,
	)

	grouped, err := ParseResourceFilter("(rock)")
	require.Nil(t, err)

	// only the inner group's own wrapper is left, the outer group adds
	// none because it holds a single filter after the empties are dropped
	assert.Equal(t,
		&ResourceFilterOr{
			Filters: []ResourceFilter{
				&ResourceFilterName{
					Name:      "rock",
					MatchCase: FilterMatchCaseInsensitive,
					MatchType: FilterMatchFuzzy,
				},
			},
		},
		grouped,
	)

	// and it matches identically to the bare name
	tags := structures.NewPackageTags()
	for _, fi := range groupCollapseSamples() {
		assert.Equal(t, bare.Matches(fi, tags), grouped.Matches(fi, tags),
			"%s must match like the bare filter", fi.ResPath)
	}
}

// TestParseFilterAllEmptyGroupMatchesNothingExtra checks that a group whose
// filters are all empty parses without error, collapses to the empty filter
// and contributes no matches of its own when it is combined with a real
// filter.
func TestParseFilterAllEmptyGroupMatchesNothingExtra(t *testing.T) {
	empty, err := ParseResourceFilter("")
	require.Nil(t, err)

	tags := structures.NewPackageTags()
	for _, str := range []string{"()", "(&)", "(())", "( )"} {
		grouped, err := ParseResourceFilter(str)
		if !assert.Nil(t, err, "%q must parse without error", str) {
			continue
		}
		_, isEmpty := grouped.(*ResourceFilterEmpty)
		assert.True(t, isEmpty, "%q must collapse to the empty filter", str)

		// the plain empty filter matches everything, so the group must
		// behave exactly like it — matching nothing extra
		for _, fi := range groupCollapseSamples() {
			assert.Equal(t, empty.Matches(fi, tags), grouped.Matches(fi, tags),
				"%q must match nothing extra for %s", str, fi.ResPath)
		}
	}

	// combined with a real filter the all-empty group adds nothing: the
	// result is the real filter alone
	bare, err := ParseResourceFilter("rock")
	require.Nil(t, err)
	for _, str := range []string{"rock ()", "rock &()"} {
		combined, err := ParseResourceFilter(str)
		if !assert.Nil(t, err, "%q must parse without error", str) {
			continue
		}
		assert.Equal(t, bare, combined, "%q must match nothing extra beyond %q", str, "rock")
	}
}

// TestParseFilterNestedGroupCollapses checks that each pair of parentheses
// contributes exactly one group: the outermost group of "((a))" collapses to
// its single filter instead of wrapping it in yet another layer.
func TestParseFilterNestedGroupCollapses(t *testing.T) {
	bare, err := ParseResourceFilter("a")
	require.Nil(t, err)
	require.Equal(t,
		&ResourceFilterName{
			Name:      "a",
			MatchCase: FilterMatchCaseInsensitive,
			MatchType: FilterMatchFuzzy,
		},
		bare,
	)

	nested, err := ParseResourceFilter("((a))")
	require.Nil(t, err)

	// one Or per pair of parentheses, no extra wrapper around the
	// outermost group's single filter
	assert.Equal(t,
		&ResourceFilterOr{
			Filters: []ResourceFilter{
				&ResourceFilterOr{
					Filters: []ResourceFilter{
						&ResourceFilterName{
							Name:      "a",
							MatchCase: FilterMatchCaseInsensitive,
							MatchType: FilterMatchFuzzy,
						},
					},
				},
			},
		},
		nested,
	)

	// behaviourally it is just "a"
	tags := structures.NewPackageTags()
	for _, fi := range groupCollapseSamples() {
		assert.Equal(t, bare.Matches(fi, tags), nested.Matches(fi, tags),
			"%s must match like the bare filter", fi.ResPath)
	}
}

// TestParseFilterGroupKeepsOrSemantics checks that parenthesising two names
// keeps the OR semantics of the unparenthesised filter: the group matches a
// resource that matches either of the names, not only one that matches both.
func TestParseFilterGroupKeepsOrSemantics(t *testing.T) {
	unparenthesized, err := ParseResourceFilter("rock tree")
	require.Nil(t, err)
	grouped, err := ParseResourceFilter("(rock tree)")
	require.Nil(t, err)

	// the group is an OR group, so the parenthesised filter is the same
	// filter as the names written next to each other
	assert.Equal(t, unparenthesized, grouped)
	assert.Equal(t,
		&ResourceFilterOr{
			Filters: []ResourceFilter{
				&ResourceFilterName{
					Name:      "rock",
					MatchCase: FilterMatchCaseInsensitive,
					MatchType: FilterMatchFuzzy,
				},
				&ResourceFilterName{
					Name:      "tree",
					MatchCase: FilterMatchCaseInsensitive,
					MatchType: FilterMatchFuzzy,
				},
			},
		},
		grouped,
	)

	// a resource that matches only one of the two names matches the
	// filter, one that matches neither does not
	tags := structures.NewPackageTags()
	rock := &structures.FileInfo{ResPath: "res://packs/testpack/textures/objects/Rock.png"}
	tree := &structures.FileInfo{ResPath: "res://packs/testpack/textures/objects/Tree.png"}
	stone := &structures.FileInfo{ResPath: "res://packs/testpack/textures/walls/Stone.png"}
	assert.True(t, grouped.Matches(rock, tags), "a resource matching only %q must match", "rock")
	assert.True(t, grouped.Matches(tree, tags), "a resource matching only %q must match", "tree")
	assert.False(t, grouped.Matches(stone, tags), "a resource matching neither must not match")
}
