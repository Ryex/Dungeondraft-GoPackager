package gui

import (
	"slices"
	"strings"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	log "github.com/sirupsen/logrus"
)

type TagFilter interface {
	Apply(tags *structures.Set[string]) bool
	ApplyToTag(tag string) bool
}

type TagFilterEmpty struct{}

func (tf TagFilterEmpty) Apply(tags *structures.Set[string]) bool {
	return true
}

func (tf TagFilterEmpty) ApplyToTag(tag string) bool {
	return true
}

type TagFilterTag struct {
	tag string
}

func (tf TagFilterTag) Apply(tags *structures.Set[string]) bool {
	ret := tags.Has(tf.tag)
	log.Tracef("set %#v has %s = %t", tags, tf.tag, ret)
	return ret
}

func (tf TagFilterTag) ApplyToTag(tag string) bool {
	return tag == tf.tag
}

type TagFilterFuzzyTag struct {
	part string
}

func (tf TagFilterFuzzyTag) Apply(tags *structures.Set[string]) bool {
	ret := utils.Any(tags.Values(), func(tag string) bool {
		return strings.Contains(strings.ToLower(tag), strings.ToLower(tf.part))
	})
	log.Tracef("set %#v fuzzy has %s = %t", tags, tf.part, ret)
	return ret
}

func (tf TagFilterFuzzyTag) ApplyToTag(tag string) bool {
	return strings.Contains(strings.ToLower(tag), strings.ToLower(tf.part))
}

type TagFilterAnd struct {
	filters []TagFilter
}

func (tf TagFilterAnd) Apply(tags *structures.Set[string]) bool {
	return utils.All(slices.Values(tf.filters), func(filter TagFilter) bool {
		return filter.Apply(tags)
	})
}

func (tf TagFilterAnd) ApplyToTag(tag string) bool {
	return utils.All(slices.Values(tf.filters), func(filter TagFilter) bool {
		return filter.ApplyToTag(tag)
	})
}

type TagFilterOr struct {
	filters []TagFilter
}

func (tf TagFilterOr) Apply(tags *structures.Set[string]) bool {
	return utils.Any(slices.Values(tf.filters), func(filter TagFilter) bool {
		return filter.Apply(tags)
	})
}

func (tf TagFilterOr) ApplyToTag(tag string) bool {
	return utils.Any(slices.Values(tf.filters), func(filter TagFilter) bool {
		return filter.ApplyToTag(tag)
	})
}

type TagFilterNot struct {
	filter TagFilter
}

func (tf TagFilterNot) Apply(tags *structures.Set[string]) bool {
	return !tf.filter.Apply(tags)
}
func (tf TagFilterNot) ApplyToTag(tag string) bool {
	return !tf.filter.ApplyToTag(tag)
}

func ParseTagFilter(s string) (filter TagFilter) {
	var args []string
	var current string
	inQuote := false
	quoteChar := '"'

	for i, char := range s {
		switch char {
		case ' ', '\t':
			if inQuote {
				current += string(char)
			} else if current != "" {
				args = append(args, current)
				current = ""

			}
		case '"', '\'':
			if inQuote && char == quoteChar {
				inQuote = false
				args = append(args, current)
				current = ""
			} else if !inQuote {
				inQuote = true
				quoteChar = char
			} else {
				current += string(char)
			}
		case '\\':
			if len(s) > i {
				nextChar := s[i+1]
				if nextChar == '"' || nextChar == '\'' || nextChar == '\\' {
					current += string(nextChar)
				} else {
					current += string(char)
				}
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}
	if current != "" {
		args = append(args, current)
	}

	lenArgs := len(args)
	if lenArgs < 1 {
		filter = TagFilterEmpty{}
	} else if lenArgs == 1 {
		arg := args[0]
		fuzzy := true
		if after, ok :=strings.CutPrefix(arg, "%"); ok  {
			arg = after
			fuzzy = false
		}
		if strings.TrimSpace(arg) == "" {
			filter = TagFilterEmpty{}
		} else if fuzzy {
			filter = TagFilterFuzzyTag{arg}
		} else {
			filter = TagFilterTag{arg}
		}
	} else {
		var filters []TagFilter
		useAnd := false
		if args[0] == "&" {
			useAnd = true
		}
		for _, arg := range args {
			invert := false
			if after, ok :=strings.CutPrefix(arg, "not:"); ok  {
				arg = after
				invert = true
			}
			fuzzy := true
			if after, ok :=strings.CutPrefix(arg, "%"); ok  {
				arg = after
				fuzzy = false
			}
			var f TagFilter

			if strings.TrimSpace(arg) == "" {
				f = TagFilterEmpty{}
			} else if fuzzy {
				f = TagFilterFuzzyTag{arg}
			} else {
				f = TagFilterTag{arg}
			}

			if invert && strings.TrimSpace(arg) != "" {
				f = TagFilterNot{f}
			}

			filters = append(filters, f)
		}
		if useAnd {
			filter = TagFilterAnd{filters}
		} else {
			filter = TagFilterOr{filters}
		}
	}
	return
}
