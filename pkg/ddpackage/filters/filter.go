// Package filter provides composeable resource filters
package filter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

type ResourceFilter interface {
	Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool
}

type ResourceFilterEmpty struct{}

func (rf ResourceFilterEmpty) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	return true
}

type FilterMatchType int

const (
	FilterMatchFuzzy FilterMatchType = iota
	FilterMatchBegin
	FilterMatchEnd
	FilterMatchExact
)

type FilterMatchCase int

const (
	FilterMatchCaseInsensitive FilterMatchCase = iota
	FilterMatchCaseSensitive
)

type ResourceFilterName struct {
	Name      string
	MatchType FilterMatchType
	MatchCase FilterMatchCase
}

func (rf ResourceFilterName) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	name, _ := strings.CutSuffix(filepath.Base(fi.ResPath), filepath.Ext(fi.ResPath))
	match := rf.Name
	if rf.MatchCase == FilterMatchCaseInsensitive {
		name = strings.ToLower(name)
		match = strings.ToLower(match)
	}
	switch rf.MatchType {
	case FilterMatchFuzzy:
		return strings.Contains(name, match)
	case FilterMatchBegin:
		return strings.HasPrefix(name, match)
	case FilterMatchEnd:
		return strings.HasSuffix(name, match)
	case FilterMatchExact:
		fallthrough
	default:
		return name == match
	}
}

type ResourceFilterTag struct {
	Tag       string
	MatchType FilterMatchType
	MatchCase FilterMatchCase
}

func (rf ResourceFilterTag) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	resourceTags := tags.TagsFor(fi.CalcRelPath())
	match := rf.Tag
	switch rf.MatchType {
	case FilterMatchFuzzy:
		if rf.MatchCase == FilterMatchCaseInsensitive {
			return resourceTags.Any(func(tag string) bool {
				return strings.Contains(strings.ToLower(tag), match)
			})
		}
		return resourceTags.Any(func(tag string) bool {
			return strings.Contains(tag, match)
		})
	case FilterMatchBegin:
		if rf.MatchCase == FilterMatchCaseInsensitive {
			return resourceTags.Any(func(tag string) bool {
				return strings.HasPrefix(strings.ToLower(tag), match)
			})
		}
		return resourceTags.Any(func(tag string) bool {
			return strings.HasPrefix(tag, match)
		})
	case FilterMatchEnd:
		if rf.MatchCase == FilterMatchCaseInsensitive {
			return resourceTags.Any(func(tag string) bool {
				return strings.HasSuffix(strings.ToLower(tag), match)
			})
		}
		return resourceTags.Any(func(tag string) bool {
			return strings.HasSuffix(tag, match)
		})
	case FilterMatchExact:
		fallthrough
	default:
		return resourceTags.Has(match)
	}
}

type ResourceFilterRelPathGlob struct {
	RelPathGlobRegexp *regexp.Regexp
}

func (rf ResourceFilterRelPathGlob) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	if rf.RelPathGlobRegexp == nil {
		return false
	}
	return rf.RelPathGlobRegexp.MatchString(fi.CalcRelPath())
}

func NewRelPathGlobFilter(pat string) (filter ResourceFilterRelPathGlob, err error) {
	filter.RelPathGlobRegexp, err = structures.GlobToRelPathRegexp(pat)
	return
}

type ResourceFilterAnd struct {
	Filters []ResourceFilter
}

func (rf ResourceFilterAnd) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	return utils.All(slices.Values(rf.Filters), func(filter ResourceFilter) bool {
		return filter.Matches(fi, tags)
	})
}

type ResourceFilterOr struct {
	Filters []ResourceFilter
}

func (rf ResourceFilterOr) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	return utils.Any(slices.Values(rf.Filters), func(filter ResourceFilter) bool {
		return filter.Matches(fi, tags)
	})
}

type ResourceFilterNot struct {
	Filter ResourceFilter
}

func (rf ResourceFilterNot) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	return !rf.Filter.Matches(fi, tags)
}

type ResourceFilterPredicate struct {
	filter func(*structures.FileInfo, *structures.PackageTags) bool
	Name   string
}

func (rf ResourceFilterPredicate) Matches(fi *structures.FileInfo, tags *structures.PackageTags) bool {
	return rf.filter(fi, tags)
}

var ResourceFilterIsTexture ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsTexture()
	},
	Name: "IsTexture",
}

var ResourceFilterIsObject ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsObject()
	},
	Name: "IsObject",
}

var ResourceFilterIsMetadata ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsMetadata()
	},
	Name: "IsMetadata",
}

var ResourceFilterIsData ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsData()
	},
	Name: "IsData",
}

var ResourceFilterIsWall ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsWall()
	},
	Name: "IsWall",
}

var ResourceFilterIsWallData ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsWallData()
	},
	Name: "IsWallData",
}

var ResourceFilterIsTileset ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsTileset()
	},
	Name: "IsTileset",
}

var ResourceFilterIsTilesetData ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsTilesetData()
	},
	Name: "IsTilesetData",
}

var ResourceFilterIsThumbnail ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsThumbnail()
	},
	Name: "IsThumbnail",
}

var ResourceFilterIsCave ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsCave()
	},
	Name: "IsCave",
}

var ResourceFilterIsLight ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsLight()
	},
	Name: "IsLight",
}

var ResourceFilterIsMaterial ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsMaterial()
	},
	Name: "IsMaterial",
}

var ResourceFilterIsPath ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsPath()
	},
	Name: "IsPath",
}

var ResourceFilterIsPattern ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsPattern()
	},
	Name: "IsPattern",
}

var ResourceFilterIsPortal ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsPortal()
	},
	Name: "IsPortal",
}

var ResourceFilterIsTerrain ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsTerrain()
	},
	Name: "IsTerrain",
}

var ResourceFilterIsTaggable ResourceFilterPredicate = ResourceFilterPredicate{
	filter: func(fi *structures.FileInfo, tags *structures.PackageTags) bool {
		return fi.IsTaggable()
	},
	Name: "IsTaggable",
}

var PredicateFilters = map[string]ResourceFilter{
	ResourceFilterIsTexture.Name:     ResourceFilterIsTexture,
	ResourceFilterIsObject.Name:      ResourceFilterIsObject,
	ResourceFilterIsMetadata.Name:    ResourceFilterIsMetadata,
	ResourceFilterIsData.Name:        ResourceFilterIsData,
	ResourceFilterIsWall.Name:        ResourceFilterIsWall,
	ResourceFilterIsWallData.Name:    ResourceFilterIsWallData,
	ResourceFilterIsTileset.Name:     ResourceFilterIsTileset,
	ResourceFilterIsTilesetData.Name: ResourceFilterIsTilesetData,
	ResourceFilterIsThumbnail.Name:   ResourceFilterIsThumbnail,
	ResourceFilterIsCave.Name:        ResourceFilterIsCave,
	ResourceFilterIsLight.Name:       ResourceFilterIsLight,
	ResourceFilterIsMaterial.Name:    ResourceFilterIsMaterial,
	ResourceFilterIsPath.Name:        ResourceFilterIsPath,
	ResourceFilterIsPattern.Name:     ResourceFilterIsPattern,
	ResourceFilterIsPortal.Name:      ResourceFilterIsPortal,
	ResourceFilterIsTerrain.Name:     ResourceFilterIsTerrain,
	ResourceFilterIsTaggable.Name:    ResourceFilterIsTaggable,
}

type ParseState int

const (
	ParseStateStart ParseState = iota
	ParseStateQuote
	ParseStateExcape
	ParseStateTag
	ParseStateName
	ParseStateFilterGlob
	ParseStateFilterName
)

type ResourceFilterGroupType int

const (
	AndGroup ResourceFilterGroupType = iota
	OrGroup
)

type ResourceFilterGroup struct {
	ty      ResourceFilterGroupType
	filters []ResourceFilter
}

func (fg ResourceFilterGroup) finish() ResourceFilter {
	// filter out empty groups
	fg.filters = slices.Collect(utils.Filter(slices.Values(fg.filters), func(rf ResourceFilter) bool {
		_, ok := rf.(*ResourceFilterEmpty)
		return !ok
	}))
	if len(fg.filters) == 0 {
		return &ResourceFilterEmpty{}
	}
	switch fg.ty {
	case OrGroup:
		return &ResourceFilterOr{
			Filters: fg.filters,
		}
	case AndGroup:
		return &ResourceFilterAnd{
			Filters: fg.filters,
		}
	default:
		panic("unreachable")
	}
}

type ParseContext struct {
	state      ParseState
	stateData  map[ParseState]any
	groupStack *structures.Stack[ResourceFilterGroup]
	c          rune
	index      int
	invertNext bool
}

type ParseDataQuote struct {
	quote rune
	prev  ParseState
	part  string
}
type ParseDataExcape struct {
	prev ParseState
}
type ParseDataName struct {
	name             string
	matchType        FilterMatchType
	matchCase        FilterMatchCase
	changedMatchType bool
	changedMatchCase bool
}
type ParseDataFilterName struct {
	name string
}
type ParseDataFilterGlob struct {
	pat string
}

func (ctx *ParseContext) data() any {
	return ctx.stateData[ctx.state]
}

func (ctx *ParseContext) dataFor(state ParseState) any {
	return ctx.stateData[state]
}

func (ctx *ParseContext) pushGroup() {
	ctx.groupStack.Push(ResourceFilterGroup{ty: OrGroup})
}

func (ctx *ParseContext) peekGroup() (*ResourceFilterGroup, error) {
	return ctx.groupStack.Peek()
}

func (ctx *ParseContext) popGroup() (ResourceFilterGroup, error) {
	return ctx.groupStack.Pop()
}

func (ctx *ParseContext) finishFilter() error {
	var filter ResourceFilter
Finish:
	switch ctx.state {
	case ParseStateTag:
		{
			data := ctx.data().(*ParseDataName)
			filter = &ResourceFilterTag{
				Tag:       data.name,
				MatchType: data.matchType,
				MatchCase: data.matchCase,
			}
		}
	case ParseStateName:
		{
			data := ctx.data().(*ParseDataName)
			filter = &ResourceFilterName{
				Name:      data.name,
				MatchType: data.matchType,
				MatchCase: data.matchCase,
			}
		}
	case ParseStateFilterGlob:
		{
			data := ctx.data().(*ParseDataFilterGlob)
			globRegexp, err := structures.GlobToRelPathRegexp(data.pat)
			if err != nil {
				ctx.newParseError(CausedByErr, err)
			}
			filter = &ResourceFilterRelPathGlob{
				RelPathGlobRegexp: globRegexp,
			}
		}
	case ParseStateFilterName:
		{
			data := ctx.data().(*ParseDataFilterName)
			var exists bool
			filter, exists = PredicateFilters[data.name]
			if !exists {
				return ctx.newParseError(UnknownFilterErr, data.name)
			}
		}
	case ParseStateQuote:
		ctx.finishQuote()
		ctx.state = ctx.data().(*ParseDataQuote).prev
		goto Finish // re-finish in new state
	case ParseStateExcape:
		ctx.state = ctx.data().(*ParseDataExcape).prev
		goto Finish // re-finish in new state
	case ParseStateStart:
		fallthrough
	default:
		{
			return ctx.newParseError(BaseParseStateErr)
		}
	}
	if ctx.invertNext {
		filter = &ResourceFilterNot{
			Filter: filter,
		}
	}
	group, _ := ctx.peekGroup()
	group.filters = append(group.filters, filter)
	ctx.invertNext = false
	return nil
}

func (ctx *ParseContext) finishQuote() {
	switch ctx.state {
	case ParseStateQuote:
		{
			data := ctx.data().(*ParseDataQuote)
			switch data.prev {
			case ParseStateName, ParseStateTag:
				prevData := ctx.dataFor(data.prev).(*ParseDataName)
				prevData.name += data.part
			case ParseStateFilterGlob:
				prevData := ctx.dataFor(data.prev).(*ParseDataFilterGlob)
				prevData.pat += data.part
			}
		}
	}
}

func (ctx *ParseContext) finishGroup() error {
	group, err := ctx.groupStack.Pop()
	if err != nil {
		return ctx.newParseError(NoGroupToCloseErr)
	}
	prevGroup, err := ctx.groupStack.Peek()
	if err != nil {
		return ctx.newParseError(NoGroupToAppendErr)
	}
	filter := group.finish()
	prevGroup.filters = append(prevGroup.filters, filter)
	return nil
}

func (ctx *ParseContext) finish() (ResourceFilter, error) {
	if ctx.groupStack.Size() > 1 {
		return nil, ctx.newParseError(UnclosedGroupErr)
	}
	group, err := ctx.groupStack.Pop()
	if err != nil {
		return nil, ctx.newParseError(NoGroupToCloseErr)
	}
	if len(group.filters) == 0 {
		return &ResourceFilterEmpty{}, nil
	}
	if len(group.filters) == 1 {
		return group.filters[0], nil
	}
	return group.finish(), nil
}

func (ctx ParseContext) newParseError(ty ParseErrorType, data ...any) error {
	if len(data) > 0 {
		return &ParseError{ErrType: ty, Index: ctx.index, Data: data[0]}
	}
	return &ParseError{ErrType: ty, Index: ctx.index}
}

type ParseErrorType int

const (
	NoGroupToCloseErr = iota
	NoGroupToAppendErr
	UnclosedGroupErr
	BaseParseStateErr
	CausedByErr
	UnknownFilterErr
)

type ParseError struct {
	ErrType ParseErrorType
	Index   int
	Data    any
}

func (p ParseError) Error() string {
	switch p.ErrType {
	case NoGroupToCloseErr:
		return fmt.Sprintf("filter parse error at %d: No group to close ')'", p.Index)
	case NoGroupToAppendErr:
		return fmt.Sprintf("filter parse error at %d: No group to append finished group", p.Index)
	case UnclosedGroupErr:
		return fmt.Sprintf("filter parse error at %d: unclosed group", p.Index)
	case BaseParseStateErr:
		return fmt.Sprintf("filter parse error at %d: base parse state", p.Index)
	case CausedByErr:
		return fmt.Sprintf("filter parse error at %d: caused by %s", p.Index, p.Data)
	case UnknownFilterErr:
		return fmt.Sprintf("filter parse error at %d: unknown filter name %s", p.Index, p.Data)
	}
	return fmt.Sprintf("filter parse error at %d: unknown", p.Index)
}

func (ctx *ParseContext) transition(next ParseState) error {
	// resolve prev
	switch ctx.state {
	case ParseStateQuote:
		ctx.finishQuote()
	case ParseStateTag, ParseStateName, ParseStateFilterGlob, ParseStateFilterName:
		if next != ParseStateExcape && next != ParseStateQuote {
			if err := ctx.finishFilter(); err != nil {
				return err
			}
		}
	case ParseStateStart:
		fallthrough
	case ParseStateExcape:
		fallthrough
	default:
		{
			// noaction
		}
	}
	// prep next
	d, exists := ctx.stateData[next]
	switch next {
	case ParseStateQuote:
		{
			if !exists {
				d = &ParseDataQuote{}
				ctx.stateData[next] = d
			}
			if ctx.state != ParseStateExcape {
				d.(*ParseDataQuote).quote = ctx.c
				d.(*ParseDataQuote).prev = ctx.state
			}
		}
	case ParseStateExcape:
		{
			if !exists {
				d = &ParseDataExcape{}
				ctx.stateData[next] = d
			}
			d.(*ParseDataExcape).prev = ctx.state
		}
	case ParseStateName, ParseStateTag:
		{
			if ctx.state != ParseStateQuote && ctx.state != ParseStateExcape {
				d = &ParseDataName{}
				ctx.stateData[next] = d
			}
		}
	case ParseStateFilterName:
		{
			d = &ParseDataFilterName{}
			ctx.stateData[next] = d
		}
	case ParseStateFilterGlob:
		{
			if ctx.state != ParseStateQuote && ctx.state != ParseStateExcape {
				d = &ParseDataFilterGlob{}
				ctx.stateData[next] = d
			}
		}
	default:
		{
			if !exists {
				ctx.stateData[next] = nil
			}
		}
	}
	ctx.state = next
	return nil
}

func NewParseContext() *ParseContext {
	return &ParseContext{
		state:      ParseStateStart,
		stateData:  map[ParseState]any{},
		groupStack: structures.NewStack[ResourceFilterGroup](),
	}
}

func ParseResourceFilter(s string) (ResourceFilter, error) {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	ctx := NewParseContext()
	ctx.pushGroup()
	for i, char := range s {
		group, _ := ctx.peekGroup()
		ctx.c = char
		ctx.index = i
		switch ctx.state {
		case ParseStateStart:
			{
				switch char {
				case '!':
					ctx.invertNext = true
				case '&':
					group.ty = AndGroup
				case '(':
					ctx.pushGroup()
				case '@':
					if err := ctx.transition(ParseStateFilterName); err != nil {
						return nil, err
					}
				case '#':
					if err := ctx.transition(ParseStateTag); err != nil {
						return nil, err
					}
				case '/':
					if err := ctx.transition(ParseStateFilterGlob); err != nil {
						return nil, err
					}
				case '%':
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					ctx.data().(*ParseDataName).matchCase = FilterMatchCaseSensitive
				case '=':
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					ctx.data().(*ParseDataName).matchType = FilterMatchExact
				case '^':
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					ctx.data().(*ParseDataName).matchType = FilterMatchBegin
				case '$':
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					ctx.data().(*ParseDataName).matchType = FilterMatchEnd
				case '"', '\'':
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					if err := ctx.transition(ParseStateQuote); err != nil {
						return nil, err
					}
				case '\\':
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					if err := ctx.transition(ParseStateExcape); err != nil {
						return nil, err
					}
				case ' ', '\t':
					{
						// do nothing
					}
				case ')':
					if err := ctx.finishGroup(); err != nil {
						return nil, err
					}
				default:
					if err := ctx.transition(ParseStateName); err != nil {
						return nil, err
					}
					ctx.data().(*ParseDataName).name += string(char)
				}
			}
		case ParseStateQuote:
			{
				data := ctx.data().(*ParseDataQuote)
				switch char {
				case data.quote: // same as opening quote
					if err := ctx.transition(data.prev); err != nil {
						return nil, err
					}
				case '\\':
					if err := ctx.transition(ParseStateExcape); err != nil {
						return nil, err
					}
				default:
					data.part += string(char)
				}
			}
		case ParseStateExcape:
			{
				data := ctx.data().(*ParseDataExcape)
				switch data.prev {
				case ParseStateQuote:
					{
						prevData := ctx.dataFor(data.prev).(*ParseDataQuote)
						switch char {
						case prevData.quote:
							prevData.part += string(char)
						case '\\':
							prevData.part += "\\"
						default:
							prevData.part += "\\" + string(char)
						}
					}
				case ParseStateName, ParseStateTag:
					{
						prevData := ctx.dataFor(data.prev).(*ParseDataName)
						switch char {
						case '(', ')', ' ', '\\', '"', '\'', '^', '$', '@', '#', '%':
							prevData.name += string(char)
						default:
							prevData.name += "\\" + string(char)
						}
					}
				case ParseStateFilterGlob:
					{
						prevData := ctx.dataFor(data.prev).(ParseDataFilterGlob)
						switch char {
						case '(':
							prevData.pat += "("
						case ')':
							prevData.pat += ")"
						case ' ', '\t':
							prevData.pat += string(char)
						case '\\':
							prevData.pat += "\\"
						case '"', '\'':
							prevData.pat += string(char)
						default:
							prevData.pat += "\\" + string(char)
						}
					}
				default:
					{
						return nil, ctx.newParseError(BaseParseStateErr)
					}
				}
				ctx.transition(data.prev)
			}
		case ParseStateTag, ParseStateName:
			{
				data := ctx.data().(*ParseDataName)
				if data.name == "" && (!data.changedMatchCase || !data.changedMatchType) {
					// still empty, can parse control chars
					// if we find any continue to next char
					switch char {
					case '%':
						if !data.changedMatchCase {
							data.matchCase = FilterMatchCaseSensitive
							data.changedMatchCase = true
							continue
						}
					case '=':
						if !data.changedMatchType {
							data.matchType = FilterMatchExact
							data.changedMatchType = true
							continue
						}
					case '^':
						if !data.changedMatchType {
							data.matchType = FilterMatchBegin
							data.changedMatchType = true
							continue
						}
					case '$':
						if !data.changedMatchType {
							data.matchType = FilterMatchEnd
							data.changedMatchType = true
							continue
						}
					default:
						{
							// fall through to matching non control chars
						}
					}
				}
				switch char {
				case '(':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
					ctx.pushGroup()
				case ')':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
					if err := ctx.finishGroup(); err != nil {
						return nil, err
					}
				case '"', '\'':
					if err := ctx.transition(ParseStateQuote); err != nil {
						return nil, err
					}
				case '\\':
					if err := ctx.transition(ParseStateExcape); err != nil {
						return nil, err
					}
				case ' ', '\t':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
				default:
					data.name += string(char)
				}
			}
		case ParseStateFilterGlob:
			{
				data := ctx.data().(*ParseDataFilterGlob)
				switch char {
				case '(':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
					ctx.pushGroup()
				case ')':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
					if err := ctx.finishGroup(); err != nil {
						return nil, err
					}
				case '"', '\'':
					if err := ctx.transition(ParseStateQuote); err != nil {
						return nil, err
					}
				case '\\':
					if err := ctx.transition(ParseStateExcape); err != nil {
						return nil, err
					}
				case ' ', '\t':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
				default:
					data.pat += string(char)
				}
			}
		case ParseStateFilterName:
			{
				data := ctx.data().(*ParseDataFilterName)
				switch char {
				case ' ', '\t':
					if err := ctx.transition(ParseStateStart); err != nil {
						return nil, err
					}
				default:
					data.name += string(char)
				}
			}
		}
	}
	ctx.finishFilter()
	return ctx.finish()
}
