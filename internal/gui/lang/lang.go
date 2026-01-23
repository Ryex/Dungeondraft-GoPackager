// Package lang reimpliments fyne/v2/lang with configurable runtime locale
//
// BSD 3-Clause License
//
// Copyright (C) 2018 Fyne.io developers (see AUTHORS)
// https://github.com/fyne-io/fyne/blob/master/AUTHORS
// All rights reserved.
//
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above copyright
//       notice, this list of conditions and the following disclaimer in the
//       documentation and/or other materials provided with the distribution.
//     * Neither the name of Fyne.io nor the names of its contributors may be
//       used to endorse or promote products derived from this software without
//       specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package lang

import (
	"embed"
	"encoding/json"
	"log"
	"strings"
	"text/template"

	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"fyne.io/fyne/v2"

	"golang.org/x/text/language"
)

var (
	// L is a shortcut to localize a string, similar to the gettext "_" function.
	// More info available on the `Localize` function.
	L = Localize

	// N is a shortcut to localize a string with plural forms, similar to the ngettext function.
	// More info available on the `LocalizePlural` function.
	N = LocalizePlural

	// X is a shortcut to get the localization of a string with specified key, similar to pgettext.
	// More info available on the `LocalizeKey` function.
	X = LocalizeKey

	// XN is a shortcut to get the localization plural form of a string with specified key, similar to npgettext.
	// More info available on the `LocalizePluralKey` function.
	XN = LocalizePluralKey

	Bundle    *i18n.Bundle
	Localizer *i18n.Localizer

	//go:embed translations
	translations embed.FS
	Translated   []language.Tag
)

// Localize asks the translation engine to translate a string, this behaves like the gettext "_" function.
// The string can be templated and the template data can be passed as a struct with exported fields,
// or as a map of string keys to any suitable value.
func Localize(in string, data ...any) string {
	return LocalizeKey(in, in, data...)
}

// LocalizeKey asks the translation engine for the translation with specific ID.
// If it cannot be found then the fallback will be used.
// The string can be templated and the template data can be passed as a struct with exported fields,
// or as a map of string keys to any suitable value.
func LocalizeKey(key, fallback string, data ...any) string {
	var d0 any
	if len(data) > 0 {
		d0 = data[0]
	}

	ret, err := Localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    key,
			Other: fallback,
		},
		TemplateData: d0,
	})

	if err != nil {
		fyne.LogError("Translation failure", err)
		return fallbackWithData(key, fallback, d0)
	}
	return ret
}

// LocalizePlural asks the translation engine to translate a string from one of a number of plural forms.
// This behaves like the ngettext function, with the `count` parameter determining the plurality looked up.
// The string can be templated and the template data can be passed as a struct with exported fields,
// or as a map of string keys to any suitable value.
func LocalizePlural(in string, count int, data ...any) string {
	return LocalizePluralKey(in, in, count, data...)
}

// LocalizePluralKey asks the translation engine for the translation with specific ID in plural form.
// This behaves like the npgettext function, with the `count` parameter determining the plurality looked up.
// If it cannot be found then the fallback will be used.
// The string can be templated and the template data can be passed as a struct with exported fields,
// or as a map of string keys to any suitable value.
func LocalizePluralKey(key, fallback string, count int, data ...any) string {
	var d0 any
	if len(data) > 0 {
		d0 = data[0]
	}

	ret, err := Localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    key,
			Other: fallback,
		},
		PluralCount:  count,
		TemplateData: d0,
	})

	if err != nil {
		fyne.LogError("Translation failure", err)
		return fallbackWithData(key, fallback, d0)
	}
	return ret
}

// AddTranslations allows an app to load a bundle of translations.
// The language that this relates to will be inferred from the resource name, for example "fr.json".
// The data should be in json format.
func AddTranslations(r fyne.Resource) error {
	defer updateLocalizer()
	return addLanguage(r.Content(), r.Name())
}

// AddTranslationsForLocale allows an app to load a bundle of translations for a specified locale.
// The data should be in json format.
func AddTranslationsForLocale(data []byte, l fyne.Locale) error {
	defer updateLocalizer()
	return addLanguage(data, l.String()+".json")
}

// AddTranslationsFS supports adding all translations in one calling using an `embed.FS` setup.
// The `dir` parameter specifies the name or path of the directory containing translation files
// inside this embedded filesystem.
// Each file should be a json file with the name following pattern [prefix.]lang.json.
func AddTranslationsFS(fs embed.FS, dir string) (retErr error) {
	files, err := fs.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		name := f.Name()
		data, err := fs.ReadFile(dir + "/" + name)
		if err != nil {
			if retErr == nil {
				retErr = err
			}
			continue
		}

		err = addLanguage(data, name)
		if err != nil {
			if retErr == nil {
				retErr = err
			}
			continue
		}
	}

	updateLocalizer()

	return retErr
}

func addLanguage(data []byte, name string) error {
	f, err := Bundle.ParseMessageFileBytes(data, name)
	if err != nil {
		return err
	}

	Translated = append(Translated, f.Tag)
	return nil
}

func init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	Translated = []language.Tag{language.Make("en")} // the first item in this list will be the fallback if none match
	err := AddTranslationsFS(translations, "translations")
	if err != nil {
		fyne.LogError("Error occurred loading built-in translations", err)
	}
}

func fallbackWithData(key, fallback string, data any) string {
	t, err := template.New(key).Parse(fallback)
	if err != nil {
		log.Println("Could not parse fallback template")
		return fallback
	}
	str := &strings.Builder{}
	_ = t.Execute(str, data)
	return str.String()
}

// SetupLang is A utility for setting up languages 
func SetupLang(lang string) {
	Localizer = i18n.NewLocalizer(Bundle, lang)
}

// updateLocalizer Finds the closest translation from the user's locale list and sets it up
func updateLocalizer() {
	all, err := locale.GetLocales()
	if err != nil {
		fyne.LogError("Failed to load user locales", err)
		all = []string{"en"}
	}
	str := ClosestSupportedLocale(all).LanguageString()
	SetupLang(str)
	Localizer = i18n.NewLocalizer(Bundle, str)
}

// SystemLocale returns the primary locale on the current system.
// This may refer to a language that Fyne does not have translations for.
func SystemLocale() fyne.Locale {
	loc, err := locale.GetLocale()
	if err != nil {
		fyne.LogError("Failed to look up user locale", err)
	}
	if len(loc) < 2 {
		loc = "en"
	}

	tag, err := language.Parse(loc)
	if err != nil {
		fyne.LogError("Error parsing user locale "+loc, err)
	}
	return LocaleFromTag(tag)
}

func ClosestSupportedLocale(locs []string) fyne.Locale {
	matcher := language.NewMatcher(Translated)

	tags := make([]language.Tag, len(locs))
	for i, loc := range locs {
		tag, err := language.Parse(loc)
		if err != nil {
			fyne.LogError("Error parsing user locale "+loc, err)
		}
		tags[i] = tag
	}
	best, _, _ := matcher.Match(tags...)
	return LocaleFromTag(best)
}

func LocaleFromTag(in language.Tag) fyne.Locale {
	b, _ := in.Base()
	r, _ := in.Region()
	s, _ := in.Script()

	return fyne.Locale(b.String() + "-" + r.String() + "-" + s.String())
}
