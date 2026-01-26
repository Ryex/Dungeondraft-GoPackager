package filter

import (
	"testing"

	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, filter, &ResourceFilterAnd{
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
