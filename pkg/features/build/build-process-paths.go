// Copyright (c) 2024  The Go-Enjin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/maruel/natural"

	"github.com/go-corelibs/maps"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/request/argv"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) ProcessGroupPath(groupChar string, w http.ResponseWriter, r *http.Request) {

	// log.WarnF("hit words group: %v", groupChar)
	var buildingPage feature.Page
	if buildingPage = f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!build/{key}"); buildingPage == nil {
		log.ErrorRF(r, "error build page not found: !build/{key}")
		f.Enjin.ServeInternalServerError(w, r)
		return
	}

	// first words starting with groupChar...
	firstWords := f.dbh.GetFirstWords(string(groupChar[0]))
	numFirstWords := len(firstWords)
	var firstLettersLen int
	if numFirstWords < 1000 {
		firstLettersLen = 2
	} else {
		firstLettersLen = 3
	}

	firstWordGrouped := make(map[string][]string)
	for _, firstWord := range firstWords {
		wordKey := quote.GetFirstCharacters(firstLettersLen, firstWord)
		firstWordGrouped[wordKey] = append(firstWordGrouped[wordKey], firstWord)
	}

	firstWordGroups := make([]*quote.WordGroup, 0)
	for _, wordKey := range maps.SortedKeys(firstWordGrouped) {
		sort.Sort(natural.StringSlice(firstWordGrouped[wordKey]))
		firstWordGroups = append(firstWordGroups, &quote.WordGroup{
			Key:   wordKey,
			Words: firstWordGrouped[wordKey],
		})
	}

	buildingPage.SetSlugUrl("/build/" + groupChar)
	buildingPage.Context().SetSpecific("FirstWordGroups", firstWordGroups)

	if err := f.Enjin.ServePage(buildingPage, w, r); err != nil {
		log.ErrorF("error serving words listing page: %v", err)
	}

	return
}

func (f *CFeature) ProcessPagePath(requestedPath string, w http.ResponseWriter, r *http.Request) {

	var buildPage feature.Page
	if buildPage = f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!b/{key}"); buildPage == nil {
		log.ErrorRF(r, "error build page not found: !b/{key}")
		f.Enjin.ServeInternalServerError(w, r)
		return
	}

	reqArgv := argv.Get(r)
	var buildingPath string

	//log.WarnF("hit: %v", m)

	if v, err := url.PathUnescape(requestedPath); err != nil {
		reqArgv.Path = "/build/"
		log.WarnF("redirecting %v due to error unescaping url path: %v", reqArgv.Path, err)
		f.Enjin.ServeRedirect(reqArgv.String(), w, r)
		return
	} else {
		buildingPath = strings.ToLower(v)
	}

	var buildPathLinks []*quote.WordLink
	var rebuiltPath, indexedPath, builtSentence string

	indexedPath = f.dbh.GetIndexedBuilderKey(buildingPath)
	rebuiltPath = f.dbh.GetHumanBuilderKey(indexedPath)
	builtSentence = strings.ReplaceAll(rebuiltPath, "-", " ")
	inputs := strings.Split(rebuiltPath, "-")
	var wordCount int
	if wordCount = len(inputs); wordCount == 0 {
		reqArgv.Path = "/build/"
		f.Enjin.ServeRedirect(reqArgv.String(), w, r)
		return
	}

	wordList, _, lookupFlat, lookupWord := f.dbh.GetWordInfoFrom(inputs...)
	var flatList []string
	for idx, word := range wordList {
		if word == "" {
			continue // inputs[idx] does not exist in the database!
		}
		if flat, present := lookupFlat[word]; present {
			flatList = append(flatList, flat)
		} else if wrd, ok := lookupWord[word]; ok {
			flatList = append(flatList, lookupFlat[wrd])
			wordList[idx] = wrd // correct the word list?
		} else {
			//panic("this is possible, user specified a URL with one or more unknown words")
		}
	}

	for idx, word := range wordList {
		buildPathLinks = append(buildPathLinks, &quote.WordLink{
			Path: strings.Join(flatList[:idx+1], "-"),
			Word: word,
		})
	}

	// log.WarnF("rebuilt=%v, index=%v, sentence=%v", rebuiltPath, indexedPath, builtSentence)

	nextWordsLookup := make(map[string]*quote.WordGroup)
	var nextWordsList []string
	if wordCount == 1 {
		indexList := strings.Split(indexedPath, "-")
		first, _ := strconv.Atoi(indexList[0])
		nextWordsList = f.dbh.GetSecondWords(string(wordList[0][0]), first)
	} else {
		nextWordsList = f.dbh.GetNextBuilderKeyWords(indexedPath)
	}
	numNextWords := len(nextWordsList)

	if numNextWords <= 25 {
		nextWordsLookup[""] = &quote.WordGroup{
			Key:   "",
			Words: nextWordsList,
		}
	} else {
		for _, word := range nextWordsList {
			groupKey := quote.GetFirstCharacters(1, word)
			if _, exists := nextWordsLookup[groupKey]; !exists {
				nextWordsLookup[groupKey] = &quote.WordGroup{
					Key: groupKey,
				}
			}
			nextWordsLookup[groupKey].Words = append(nextWordsLookup[groupKey].Words, word)
		}
	}
	var nextWordsGrouped []*quote.WordGroup
	for _, groupKey := range maps.SortedKeys(nextWordsLookup) {
		if len(nextWordsLookup[groupKey].Words) > 0 {
			nextWordsGrouped = append(nextWordsGrouped, nextWordsLookup[groupKey])
		}
	}

	_, builtQuotes := f.dbh.GetBuilderKeyQuotes(indexedPath)
	numBuiltQuotes := len(builtQuotes)
	if numBuiltQuotes == 1 {
		buildPage.Context().SetSpecific("Title", `Quoted.FYI: Built - `+builtSentence)
	} else {
		buildPage.Context().SetSpecific("Title", `Quoted.FYI: Building - `+builtSentence)
	}
	buildPage.SetSlugUrl("/b/" + rebuiltPath)
	buildPage.Context().SetSpecific("BuildPath", rebuiltPath)
	buildPage.Context().SetSpecific("BuildPathLinks", buildPathLinks)
	buildPage.Context().SetSpecific("NextWordGroups", nextWordsGrouped)
	buildPage.Context().SetSpecific("NumNextWords", numNextWords)
	buildPage.Context().SetSpecific("NumNextWordGroups", len(nextWordsGrouped))
	buildPage.Context().SetSpecific("BuiltQuotes", builtQuotes)
	buildPage.Context().SetSpecific("NumBuiltQuotes", numBuiltQuotes)
	if err := f.Enjin.ServePage(buildPage, w, r); err != nil {
		log.ErrorF("error serving words listing page: %v", err)
	}
	return
}
