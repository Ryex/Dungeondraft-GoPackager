package structures

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Package stores package information for the pack.json
type Package struct {
	Name           string               `json:"name"`
	ID             string               `json:"id"`
	Version        string               `json:"version"`
	Author         string               `json:"author"`
	KeywordsRaw    string               `json:"keywords"`
	Keywords       []string             `json:"-"`
	Allow3rdParty  *bool                `json:"allow_3rd_party_mapping_software_to_read,omitempty"`
	ColorOverrides CustomColorOverrides `json:"custom_color_overrides,omitempty"`
}

type CustomColorOverrides struct {
	Enabled       bool    `json:"enabled"`
	MinRedness    float64 `json:"min_redness"`
	MinSaturation float64 `json:"min_saturation"`
	RedTolerance  float64 `json:"red_tolerance"`
}

func (o *CustomColorOverrides) String() string {
	return fmt.Sprintf("%v", *o)
}

func (o *CustomColorOverrides) Set(value string) error {
	parts := strings.Split(value, ",")
	defaultErr := errors.New("Color Overrides format is <redness>,<saturation>,<red_tolerance>")
	if len(parts) != 3 {
		return defaultErr
	}
	if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
		o.MinRedness = v
	} else {
		return defaultErr
	}
	if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
		o.MinSaturation = v
	} else {
		return defaultErr
	}
	if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
		o.RedTolerance = v
	} else {
		return defaultErr
	}
	o.Enabled = true
	return nil
}
