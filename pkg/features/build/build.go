// Copyright (c) 2022  The Go-Enjin Authors
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
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-enjin/golang-org-x-text/language"
	"github.com/maruel/natural"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/pkg/context"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
	"github.com/go-enjin/be/pkg/page"
	"github.com/go-enjin/be/pkg/pagecache"
	"github.com/go-enjin/be/pkg/regexps"
	"github.com/go-enjin/be/pkg/request/argv"
	"github.com/go-enjin/be/pkg/theme"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var RxFirstWidInPath = regexp.MustCompile(`^(\d+)`)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "PagesQuoteBuild"

type Feature interface {
	feature.Middleware
	feature.PageTypeProcessor
	pagecache.PageIndexFeature
}

type CFeature struct {
	feature.CMiddleware

	cli   *cli.Context
	enjin feature.Internals

	kwp   pagecache.KeywordProvider
	theme *theme.Theme

	knownWords  []string
	lookupWords map[string]int
	knownPaths  map[string][]int
	lookupStubs map[int]*pagecache.Stub

	lastStubIdx       int
	lastKnownWordsIdx int

	sync.RWMutex
}

type MakeFeature interface {
	Make() Feature
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	return f
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CMiddleware.Init(this)
	f.kwp = nil
	f.knownPaths = make(map[string][]int)
	f.lookupStubs = make(map[int]*pagecache.Stub)
	f.lookupWords = make(map[string]int)
}

func (f *CFeature) Tag() (tag feature.Tag) {
	tag = Tag
	return
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.enjin = enjin
	if t, err := f.enjin.GetTheme(); err != nil {
		log.FatalF("error getting enjin theme: %v", err)
	} else {
		f.theme = t
	}
	for _, feat := range f.enjin.Features() {
		if kwp, ok := feat.(pagecache.KeywordProvider); ok {
			f.kwp = kwp
			break
		}
	}
	if f.kwp == nil {
		log.FatalF("%v requires a pagecache.KeywordProvider feature to be present", f.Tag())
	}
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	f.cli = ctx
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote words middleware")
	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// TODO: redirect /w/ -> /words/, etc
			switch {
			case f.ProcessPagePath(w, r):
				return
			case f.ProcessGroupPath(w, r):
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (f *CFeature) FilterPageContext(themeCtx, pageCtx context.Context, r *http.Request) (themeOut context.Context) {
	themeOut = themeCtx
	if pgType, ok := pageCtx.Get("Type").(string); ok {
		if pgType == "quote" {
			var quoteBuilderKey string
			if content, ok := pageCtx.Get("Content").(string); ok {
				keywords := regexps.RxKeywords.FindAllString(strings.ToLower(content), -1)
				var swids []string
				for _, word := range keywords {
					if wid, ok := f.lookupWords[word]; ok {
						swids = append(swids, fmt.Sprintf("%d", wid))
					}
				}
				for i := len(swids) - 1; i >= 0; i-- {
					if i == 0 {
						quoteBuilderKey = keywords[0]
					} else {
						indexPath := strings.Join(swids[:i], "-")
						if found := f.getBuiltQuotes(indexPath); len(found) > 1 {
							quoteBuilderKey = strings.Join(keywords[:i], "-")
							// log.WarnF("found quote builder key: %v", keywords[:i])
							break
						}
					}
				}
			}
			themeOut.SetSpecific("QuoteBuilderKey", quoteBuilderKey)
			// log.WarnF("quote builder key: %v", quoteBuilderKey)
		}
	}
	return
}

var RxPagePath = regexp.MustCompile(`^/b/([^/]+?)/??$`)

func (f *CFeature) ProcessPagePath(w http.ResponseWriter, r *http.Request) (processed bool) {
	switch r.URL.Path {
	case "/b", "/b/":
		reqArgv := argv.DecodeHttpRequest(r)
		reqArgv.Path = "/build/"
		f.enjin.ServeRedirect(reqArgv.String(), w, r)
		processed = true
		return
	}
	if RxPagePath.MatchString(r.URL.Path) {
		if buildPage := f.enjin.FindPage(f.enjin.SiteDefaultLanguage(), "!b/{key}"); buildPage != nil {
			reqArgv := argv.DecodeHttpRequest(r)
			m := RxPagePath.FindAllStringSubmatch(r.URL.Path, 1)
			requestedPath := m[0][1]
			buildingPath := strings.ToLower(requestedPath)

			log.WarnF("hit: %v", m)

			if v, err := url.PathUnescape(buildingPath); err != nil {
				reqArgv.Path = "/build/"
				log.WarnF("redirecting %v due to error unescaping url path: %v", reqArgv.Path, err)
				f.enjin.ServeRedirect(reqArgv.String(), w, r)
				processed = true
				return
			} else {
				buildingPath = v
			}

			var buildPathLinks []*quote.WordLink
			var rebuiltPath, indexPath, builtSentence string
			buildPathWords := strings.Split(buildingPath, "-")
			for idx, word := range buildPathWords {
				if v, ok := f.lookupWords[word]; ok {
					if idx > 0 {
						rebuiltPath += "-"
						builtSentence += " "
						indexPath += "-"
					}
					rebuiltPath += word
					builtSentence += word
					indexPath += fmt.Sprintf("%d", v)
					buildPathLinks = append(buildPathLinks, &quote.WordLink{
						Path: rebuiltPath,
						Word: word,
					})
				} else {
					reqArgv.Path = "/b/" + rebuiltPath
					log.WarnF("redirecting %v due to invalid word requested: \"%v\"", reqArgv.Path, word)
					f.enjin.ServeRedirect(reqArgv.String(), w, r)
					processed = true
					return
				}
			}
			// log.WarnF("rebuilt=%v, index=%v, sentence=%v", rebuiltPath, indexPath, builtSentence)

			// need: .BuiltQuotes[]{Url,Hash} .NextWords{Grouped{Key,Words}} .BuildPathLinks[]{Path,Word}

			nextWordsLookup := make(map[string]*quote.WordGroup)
			nextWordsList := f.getNextWords(indexPath)
			numNextWords := len(nextWordsList)
			if numNextWords < 25 {
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

			builtQuotes := f.getBuiltQuotes(indexPath)
			numBuiltQuotes := len(builtQuotes)
			if numBuiltQuotes == 1 {
				buildPage.Context.SetSpecific("Title", `Quoted.FYI: Built - `+builtSentence)
			} else {
				buildPage.Context.SetSpecific("Title", `Quoted.FYI: Building - `+builtSentence)
			}
			buildPage.SetSlugUrl("/b/" + rebuiltPath)
			buildPage.Context.SetSpecific("BuildPath", rebuiltPath)
			buildPage.Context.SetSpecific("BuildPathLinks", buildPathLinks)
			buildPage.Context.SetSpecific("NextWordGroups", nextWordsGrouped)
			buildPage.Context.SetSpecific("NumNextWordGroups", len(nextWordsGrouped))
			buildPage.Context.SetSpecific("BuiltQuotes", builtQuotes)
			buildPage.Context.SetSpecific("NumBuiltQuotes", numBuiltQuotes)
			if err := f.enjin.ServePage(buildPage, w, r); err != nil {
				log.ErrorF("error serving words listing page: %v", err)
			} else {
				processed = true
			}
		}
	}
	return
}

var RxGroupPath = regexp.MustCompile(`^/build/([a-zA-Z0-9])?/??`)

func (f *CFeature) ProcessGroupPath(w http.ResponseWriter, r *http.Request) (processed bool) {
	if RxGroupPath.MatchString(r.URL.Path) {
		m := RxGroupPath.FindAllStringSubmatch(r.URL.Path, 1)
		groupChar := strings.ToLower(m[0][1])
		// log.WarnF("hit words group: %v", groupChar)
		if buildingPage := f.enjin.FindPage(f.enjin.SiteDefaultLanguage(), "!build/{key}"); buildingPage != nil {

			// first words starting with groupChar...
			firstWords := f.getFirstWordsStartingWith(groupChar[0])
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
			// buildingPage.Context.SetSpecific("Topics", words)
			// buildingPage.Context.SetSpecific("TopicLetters", topicLetters)
			// buildingPage.Context.SetSpecific("NumTopics", len(words))
			buildingPage.Context.SetSpecific("FirstWordGroups", firstWordGroups)
			// buildingPage.Context.SetSpecific("TopicCharacter", groupChar)
			// buildingPage.Context.SetSpecific("TotalNumTopics", totalNumWords)
			if err := f.enjin.ServePage(buildingPage, w, r); err != nil {
				log.ErrorF("error serving words listing page: %v", err)
			} else {
				processed = true
			}
		}
	}
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// reqArgv := site.GetRequestArgv(r)

	switch p.Type {
	case "build":
		pg, redirect, processed, err = f.ProcessBuildPageType(r, p)
	case "builder":
		pg, redirect, processed, err = f.ProcessBuilderPageType(r, p)
	case "building":
		pg, redirect, processed, err = f.ProcessBuildingPageType(r, p)
	}

	return
}

func (f *CFeature) ProcessBuildPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit build page type: %v", p.Url)
	p.Context.SetSpecific("FirstWordFirstLetters", f.getFirstWordFirstLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) ProcessBuilderPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit builder page type: %v", p.Url)
	p.Context.SetSpecific("FirstWordFirstLetters", f.getFirstWordFirstLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) ProcessBuildingPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit building page type: %v", p.Url)
	p.Context.SetSpecific("FirstWordFirstLetters", f.getFirstWordFirstLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) parseContentKeywords(content string) (keywords []string) {
	for _, keyword := range regexps.RxKeywords.FindAllString(content, -1) {
		keyword = strings.ToLower(keyword)
		// keywords = append(keywords, keyword)
		if parts := strings.Split(keyword, "-"); len(parts) > 0 {
			keywords = append(keywords, parts...)
			// for _, part := range parts {
			// 	keywords = append(keywords, part)
			// }
		}
	}
	return
}

func (f *CFeature) AddToIndex(stub *pagecache.Stub, p *page.Page) (err error) {

	if p.Type != "quote" {
		return
	}

	f.Lock()
	defer f.Unlock()

	f.lastStubIdx += 1 // from -1, first key is 0
	f.lookupStubs[f.lastStubIdx] = stub

	var path string
	foundWords := f.parseContentKeywords(p.Content)
	for idx, keyword := range foundWords {
		if _, exists := f.lookupWords[keyword]; !exists {
			f.knownWords = append(f.knownWords, keyword)
			f.lastKnownWordsIdx = len(f.knownWords) - 1
			f.lookupWords[keyword] = f.lastKnownWordsIdx
		}
		if idx > 0 {
			path += "-"
		}
		path += strconv.Itoa(f.lookupWords[keyword])
	}

	f.knownPaths[path] = append(f.knownPaths[path], f.lastStubIdx)
	return
}

func (f *CFeature) RemoveFromIndex(tag language.Tag, file string, shasum string) {
	return
}

func (f *CFeature) getFirstWordFirstLetters() (letters []string) {
	f.RLock()
	defer f.RUnlock()
	cache := make(map[string]bool)
	for key, _ := range f.knownPaths {
		m := RxFirstWidInPath.FindAllString(key, 1)
		wid, _ := strconv.Atoi(m[0])
		if wid >= 0 && wid < len(f.knownWords) {
			word := f.knownWords[wid]
			letter := string(word[0])
			cache[letter] = true
		}
	}
	letters = maps.SortedKeys(cache)
	return
}

func (f *CFeature) getFirstWordsStartingWith(prefix uint8) (words []string) {
	f.RLock()
	defer f.RUnlock()
	cache := make(map[string]bool)
	for key, _ := range f.knownPaths {
		m := RxFirstWidInPath.FindAllString(key, 1)
		wid, _ := strconv.Atoi(m[0])
		if wid >= 0 && wid < len(f.knownWords) {
			word := f.knownWords[wid]
			if word[0] == prefix {
				cache[word] = true
			}
		}
	}
	words = maps.SortedKeys(cache)
	return
}

func (f *CFeature) getNextWords(indexPath string) (words []string) {
	f.RLock()
	defer f.RUnlock()
	foundWords := make(map[string]bool)
	indexPathLength := len(indexPath)
	prefixPath := indexPath + "-"
	for path, _ := range f.knownPaths {
		if pathLength := len(path); pathLength > indexPathLength {
			if path[:indexPathLength+1] == prefixPath {
				suffix := path[indexPathLength+1:]
				// log.WarnF("suffix=%v, path=%v, prefix=%v", suffix, path, prefixPath)
				wids := strings.Split(suffix, "-")
				if len(wids) > 0 {
					if wid, err := strconv.Atoi(wids[0]); err != nil {
						log.ErrorF("error converting wid to int: \"%v\" - %v", wids[0], err)
					} else if wid >= 0 && wid <= f.lastKnownWordsIdx {
						word := f.knownWords[wid]
						foundWords[word] = true
						// log.WarnF("suffix=%v, path=%v, prefix=%v - word: %v", suffix, path, prefixPath, word)
					} else {
						log.ErrorF("wid out of range: %v [0-%v]", wid, f.lastKnownWordsIdx)
					}
				}
			}
		}
	}
	words = maps.SortedKeys(foundWords)
	// log.WarnF("got next words: index=\"%v\", numWords=%d", indexPath, len(words))
	return
}

func (f *CFeature) getBuiltQuotes(indexPath string) (builtQuotes []*quote.Quote) {
	f.RLock()
	defer f.RUnlock()
	stubCount := 0
	stubsLookup := make(map[int]bool)
	reqPathLen := len(indexPath)
	prefixPath := indexPath + "-"
	prefixPathLen := reqPathLen + 1
	for path, ids := range f.knownPaths {
		if pathLen := len(path); pathLen >= reqPathLen {
			switch {
			case path == indexPath:
			case pathLen >= prefixPathLen && path[:prefixPathLen] == prefixPath:
			default:
				continue
			}
			if stubCount += len(ids); stubCount > 25 {
				return
			}
			// log.WarnF("found built quote:\nindexPath=%v\npath=%v\nids=%v", indexPath, path, ids)
			for _, id := range ids {
				stubsLookup[id] = true
			}
		}
	}
	for idx, _ := range stubsLookup {
		if idx >= 0 && idx <= f.lastStubIdx {
			stub := f.lookupStubs[idx]
			if pg, err := stub.Make(f.theme); err != nil {
				log.ErrorF("error making page from cache: %v - %v", stub.Source, err)
			} else if hash, ok := pg.Context.Get("QuoteHash").(string); ok {
				builtQuotes = append(builtQuotes, &quote.Quote{
					Url:  pg.Url,
					Hash: hash,
				})
			} else {
				log.ErrorF("error page missing QuoteHash: %v", pg.Url)
			}
		} else {
			log.WarnF("stub index out of bounds: %v [0-%v]", idx, f.lastStubIdx)
		}
	}
	return
}