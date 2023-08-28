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
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/maruel/natural"
	"github.com/puzpuzpuz/xsync/v2"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/golang-org-x-text/language"

	"github.com/go-enjin/be/pkg/context"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
	"github.com/go-enjin/be/pkg/regexps"
	"github.com/go-enjin/be/pkg/request/argv"
	"github.com/go-enjin/be/types/page"

	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var RxFirstWidInPath = regexp.MustCompile(`^(\d+)`)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "build-quote-pages"

type Feature interface {
	feature.Feature
	feature.UseMiddleware
	feature.PageTypeProcessor
	feature.PageIndexFeature
	feature.PageContextModifier
}

type MakeFeature interface {
	Make() Feature

	SetKeywordProvider(kwp feature.Tag) MakeFeature
}

type CFeature struct {
	feature.CFeature

	kwpTag feature.Tag
	kwp    feature.KeywordProvider
	theme  feature.Theme

	knownWords []string
	knownStubs *xsync.MapOf[int, string]
	indexPaths *xsync.MapOf[string, []int]
	builderKey *xsync.MapOf[string, string]

	lookupWords    *xsync.MapOf[string, int]
	lookupUnquoted *xsync.MapOf[int, int]

	lastStubIdx       int
	lastKnownWordsIdx int
}

func New() MakeFeature {
	return NewTagged(Tag)
}

func NewTagged(tag feature.Tag) MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = tag
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
	f.kwp = nil
	f.knownStubs = xsync.NewIntegerMapOf[int, string]()
	f.indexPaths = xsync.NewMapOf[[]int]()
	f.builderKey = xsync.NewMapOf[string]()
	f.lookupWords = xsync.NewMapOf[int]()
	f.lookupUnquoted = xsync.NewIntegerMapOf[int, int]()
}

func (f *CFeature) SetKeywordProvider(kwp feature.Tag) MakeFeature {
	f.kwpTag = kwp
	return f
}

func (f *CFeature) Make() Feature {
	if f.kwpTag == feature.NilTag {
		log.FatalDF(1, "%v feature requires .SetKeywordProvider", f.Tag())
	}
	return f
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
	if t, err := f.Enjin.GetTheme(); err != nil {
		log.FatalF("error getting enjin theme: %v", err)
	} else {
		f.theme = t
	}

	if kwpf, ok := f.Enjin.Features().Get(f.kwpTag); !ok {
		log.FatalF("%v failed to find %v feature", f.Tag(), f.kwpTag)
	} else if kwp, ok := feature.AsTyped[feature.KeywordProvider](kwpf); !ok {
		log.FatalF("%v feature is not an indexing.KeywordProvider", f.kwpTag)
	} else {
		f.kwp = kwp
	}
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	err = f.CFeature.Startup(ctx)
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote words middleware")

	// TODO: investigate precaching QuoteBuilderKey for each quote (shortest non-specific index path)
	/*
		log.DebugF("%v: caching builder key lookups", f.Tag())
		start := time.Now()
		var count, errcount int
		t := f.Enjin.MustGetTheme()
		prevGC := debug.SetGCPercent(50)
		f.knownStubs.Range(func(key int, shasum string) bool {
			if stub := f.Enjin.FindPageStub(shasum); stub == nil {
				log.ErrorF("error page stub not found: %v", shasum)
				errcount += 1
			} else if pg, err := page.NewFromPageStub(stub, t); err != nil {
				log.ErrorF("error making page from stub: %v - %v", shasum, err)
				errcount += 1
			} else {
				keywords := f.parseContentKeywords(pg.Content)
				var quoteBuilderKey, quoteHash string
				if quoteHash = pg.Context().String("QuoteHash", ""); quoteHash == "" {
					log.ErrorF("error quote hash not found: %v", shasum)
					return true
				}

				var wids []string
				for _, word := range keywords {
					wid, _ := f.lookupWords.Load(word)
					wids = append(wids, strconv.Itoa(wid))
				}
				for i := len(wids) - 1; i >= 0; i-- {
					if i == 0 {
						quoteBuilderKey = keywords[0]
					} else {
						indexPath := strings.Join(wids[:i], "-")
						if found := f.countBuiltQuotes(indexPath); found > 1 {
							quoteBuilderKey = strings.Join(keywords[:i], "-")
							// log.WarnF("found quote builder key: %v", keywords[:i])
							break
						}
					}
				}

				f.builderKey.Store(quoteHash, quoteBuilderKey)
				count += 1
			}
			return true
		})
		debug.SetGCPercent(prevGC)
		delta := time.Now().Sub(start)
		log.DebugF("%v: cached %d builder keys in %v (with %d errors)", f.Tag(), count, delta.String(), errcount)
	*/

	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// TODO: redirect /b/ -> /build/, etc
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

func (f *CFeature) findShortestBuilderKey(content string) (quoteBuilderKey string) {
	keywords := f.parseContentKeywords(content)
	var wids []string
	for _, word := range keywords {
		wid, _ := f.lookupWords.Load(word)
		wids = append(wids, strconv.Itoa(wid))
	}
	/*
		// reverse lookup
		for i := len(wids) - 1; i >= 0; i-- {
			if i == 0 {
				quoteBuilderKey = keywords[0]
			} else {
				indexPath := strings.Join(wids[:i], "-")
				if found := f.countBuiltQuotes(indexPath); found > 1 {
					quoteBuilderKey = strings.Join(keywords[:i], "-")
					// log.WarnF("found quote builder key: %v", keywords[:i])
					break
				}
			}
		}
	*/
	// forward lookup
	var prevCount int
	for i := 1; i < len(wids); i++ {
		indexPath := strings.Join(wids[:i], "-")
		if found := f.countBuiltQuotes(indexPath); found == 1 {
			quoteBuilderKey = strings.Join(keywords[:i-1], "-")
			// log.WarnF("found quote builder key: %v", keywords[:i])
			break
		} else if prevCount == 2 && found == 2 && i > 2 {
			// catch plural singularities?
			quoteBuilderKey = strings.Join(keywords[:i-2], "-")
			break
		} else {
			prevCount = found
		}
	}
	return
}

func (f *CFeature) FilterPageContext(themeCtx, pageCtx context.Context, r *http.Request) (themeOut context.Context) {
	themeOut = themeCtx
	if pgType, ok := pageCtx.Get("Type").(string); ok {
		if pgType == "quote" {
			themeOut.SetSpecific(
				"QuoteBuilderKey",
				f.findShortestBuilderKey(
					pageCtx.String("Content", ""),
				),
			)
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
		f.Enjin.ServeRedirect(reqArgv.String(), w, r)
		processed = true
		return
	}
	if RxPagePath.MatchString(r.URL.Path) {
		if buildPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!b/{key}"); buildPage != nil {
			reqArgv := argv.DecodeHttpRequest(r)
			m := RxPagePath.FindAllStringSubmatch(r.URL.Path, 1)
			requestedPath := m[0][1]
			buildingPath := strings.ToLower(requestedPath)

			//log.WarnF("hit: %v", m)

			if v, err := url.PathUnescape(buildingPath); err != nil {
				reqArgv.Path = "/build/"
				log.WarnF("redirecting %v due to error unescaping url path: %v", reqArgv.Path, err)
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				processed = true
				return
			} else {
				buildingPath = v
			}

			var buildPathLinks []*quote.WordLink
			var rebuiltPath, indexPath, builtSentence string
			buildPathWords := strings.Split(buildingPath, "-")
			for idx, word := range buildPathWords {

				if v, ok := f.lookupWords.Load(word); ok {
					if original, ok := f.lookupUnquoted.Load(v); ok {
						word = f.knownWords[original]
						v = original
					}
					if idx > 0 {
						rebuiltPath += "-"
						builtSentence += " "
						indexPath += "-"
					}
					rebuiltPath += word
					builtSentence += word
					indexPath += strconv.Itoa(v)
					buildPathLinks = append(buildPathLinks, &quote.WordLink{
						Path: rebuiltPath,
						Word: word,
					})
				} else {
					reqArgv.Path = "/b/" + rebuiltPath
					log.WarnF("redirecting %v due to invalid word requested: \"%v\"", reqArgv.Path, word)
					f.Enjin.ServeRedirect(reqArgv.String(), w, r)
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
				buildPage.Context().SetSpecific("Title", `Quoted.FYI: Built - `+builtSentence)
			} else {
				buildPage.Context().SetSpecific("Title", `Quoted.FYI: Building - `+builtSentence)
			}
			buildPage.SetSlugUrl("/b/" + rebuiltPath)
			buildPage.Context().SetSpecific("BuildPath", rebuiltPath)
			buildPage.Context().SetSpecific("BuildPathLinks", buildPathLinks)
			buildPage.Context().SetSpecific("NextWordGroups", nextWordsGrouped)
			buildPage.Context().SetSpecific("NumNextWordGroups", len(nextWordsGrouped))
			buildPage.Context().SetSpecific("BuiltQuotes", builtQuotes)
			buildPage.Context().SetSpecific("NumBuiltQuotes", numBuiltQuotes)
			if err := f.Enjin.ServePage(buildPage, w, r); err != nil {
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
		if buildingPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!build/{key}"); buildingPage != nil {

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
			// buildingPage.Context().SetSpecific("Topics", words)
			// buildingPage.Context().SetSpecific("TopicLetters", topicLetters)
			// buildingPage.Context().SetSpecific("NumTopics", len(words))
			buildingPage.Context().SetSpecific("FirstWordGroups", firstWordGroups)
			// buildingPage.Context().SetSpecific("TopicCharacter", groupChar)
			// buildingPage.Context().SetSpecific("TotalNumTopics", totalNumWords)
			if err := f.Enjin.ServePage(buildingPage, w, r); err != nil {
				log.ErrorF("error serving words listing page: %v", err)
			} else {
				processed = true
			}
		}
	}
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// reqArgv := site.GetRequestArgv(r)

	switch p.Type() {
	case "build":
		pg, redirect, processed, err = f.ProcessBuildPageType(r, p)
	case "builder":
		pg, redirect, processed, err = f.ProcessBuilderPageType(r, p)
	case "building":
		pg, redirect, processed, err = f.ProcessBuildingPageType(r, p)
	}

	return
}

func (f *CFeature) ProcessBuildPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit build page type: %v", p.Url())
	p.Context().SetSpecific("FirstWordFirstLetters", f.getFirstWordFirstLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) ProcessBuilderPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit builder page type: %v", p.Url())
	p.Context().SetSpecific("FirstWordFirstLetters", f.getFirstWordFirstLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) ProcessBuildingPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit building page type: %v", p.Url())
	p.Context().SetSpecific("FirstWordFirstLetters", f.getFirstWordFirstLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) parseContentKeywords(content string) (keywords []string) {
	for _, keyword := range regexps.RxKeywords.FindAllString(content, -1) {
		keyword = strings.ToLower(keyword)
		if parts := strings.Split(keyword, "-"); len(parts) > 0 {
			keywords = append(keywords, parts...)
		}
	}
	return
}

func (f *CFeature) AddToIndex(stub *feature.PageStub, p feature.Page) (err error) {

	if p.Type() != "quote" {
		return
	}

	f.Lock()
	defer f.Unlock()

	f.lastStubIdx += 1 // from -1, first key is 0
	f.knownStubs.Store(f.lastStubIdx, stub.Shasum)

	addLookupWord := func(keyword string) {
		if _, exists := f.lookupWords.Load(keyword); !exists {
			f.knownWords = append(f.knownWords, keyword)
			f.lastKnownWordsIdx = len(f.knownWords) - 1
			f.lookupWords.Store(keyword, f.lastKnownWordsIdx)
		}
	}

	var indexPath string
	foundWords := f.parseContentKeywords(p.Content())
	for idx, keyword := range foundWords {
		addLookupWord(keyword)
		if idx > 0 {
			indexPath += "-"
		}
		wid, _ := f.lookupWords.Load(keyword)
		indexPath += strconv.Itoa(wid)
	}
	knownPaths, _ := f.indexPaths.Load(indexPath)
	f.indexPaths.Store(indexPath, append(knownPaths, f.lastStubIdx))

	var unqPath string
	for idx, keyword := range foundWords {
		if idx > 0 {
			unqPath += "-"
		}
		if unquoted := strings.ReplaceAll(keyword, "'", "_"); keyword != unquoted {
			addLookupWord(unquoted)
			widUnq, _ := f.lookupWords.Load(unquoted)
			widKw, _ := f.lookupWords.Load(keyword)
			f.lookupUnquoted.Store(widUnq, widKw)
			unqPath += strconv.Itoa(widUnq)
		} else {
			wid, _ := f.lookupWords.Load(keyword)
			unqPath += strconv.Itoa(wid)
		}
	}
	if unqPath != indexPath {
		unqPaths, _ := f.indexPaths.Load(unqPath)
		f.indexPaths.Store(unqPath, append(unqPaths, f.lastStubIdx))
	}

	return
}

func (f *CFeature) RemoveFromIndex(tag language.Tag, file string, shasum string) {
	return
}

func (f *CFeature) getFirstWordFirstLetters() (letters []string) {
	//f.RLock()
	//defer f.RUnlock()
	cache := make(map[string]bool)
	f.indexPaths.Range(func(key string, _ []int) bool {
		m := RxFirstWidInPath.FindAllString(key, 1)
		wid, _ := strconv.Atoi(m[0])
		if wid >= 0 && wid < len(f.knownWords) {
			word := f.knownWords[wid]
			letter := string(word[0])
			cache[letter] = true
		}
		return true
	})
	letters = maps.SortedKeys(cache)
	return
}

func (f *CFeature) getFirstWordsStartingWith(prefix uint8) (words []string) {
	//f.RLock()
	//defer f.RUnlock()
	cache := make(map[string]bool)
	f.indexPaths.Range(func(key string, _ []int) bool {
		m := RxFirstWidInPath.FindAllString(key, 1)
		wid, _ := strconv.Atoi(m[0])
		if wid >= 0 && wid < len(f.knownWords) {
			if v, ok := f.lookupUnquoted.Load(wid); ok {
				wid = v
			}
			word := f.knownWords[wid]
			if word[0] == prefix {
				cache[word] = true
			}
		}
		return true
	})
	words = maps.SortedKeys(cache)
	return
}

func (f *CFeature) getNextWords(indexPath string) (words []string) {
	//f.RLock()
	//defer f.RUnlock()
	foundWords := make(map[string]bool)
	indexPathLength := len(indexPath)
	prefixPath := indexPath + "-"
	f.indexPaths.Range(func(key string, _ []int) bool {
		if pathLength := len(key); pathLength > indexPathLength {
			if key[:indexPathLength+1] == prefixPath {
				suffix := key[indexPathLength+1:]
				// log.WarnF("suffix=%v, key=%v, prefix=%v", suffix, key, prefixPath)
				wids := strings.Split(suffix, "-")
				if len(wids) > 0 {
					if wid, err := strconv.Atoi(wids[0]); err != nil {
						log.ErrorF("error converting wid to int: \"%v\" - %v", wids[0], err)
					} else if wid >= 0 && wid <= f.lastKnownWordsIdx {
						if v, ok := f.lookupUnquoted.Load(wid); ok {
							wid = v
						}
						word := f.knownWords[wid]
						foundWords[word] = true
						// log.WarnF("suffix=%v, key=%v, prefix=%v - word: %v", suffix, key, prefixPath, word)
					} else {
						log.ErrorF("wid out of range: %v [0-%v]", wid, f.lastKnownWordsIdx)
					}
				}
			}
		}
		return true
	})
	words = maps.SortedKeys(foundWords)
	// log.WarnF("got next words: index=\"%v\", numWords=%d", indexPath, len(words))
	return
}

func (f *CFeature) countBuiltQuotes(indexPath string) (count int) {
	reqPathLen := len(indexPath)
	prefixPath := indexPath + "-"
	prefixPathLen := reqPathLen + 1
	fn := func(key string, ids []int) bool {
		if pathLen := len(key); pathLen >= reqPathLen {
			switch {
			case key == indexPath:
				// exact match
			case pathLen >= prefixPathLen && key[:prefixPathLen] == prefixPath:
				// prefix match
			default:
				return true
			}
			count += len(ids)
			//return false
		}
		return true
	}
	f.indexPaths.Range(fn)
	return
}

func (f *CFeature) getBuiltQuotes(indexPath string) (builtQuotes []*quote.Quote) {
	//f.RLock()
	//defer f.RUnlock()
	stubCount := 0
	stubsLookup := make(map[int]bool)
	reqPathLen := len(indexPath)
	prefixPath := indexPath + "-"
	prefixPathLen := reqPathLen + 1

	var done bool
	f.indexPaths.Range(func(key string, ids []int) bool {
		if pathLen := len(key); pathLen >= reqPathLen {
			switch {
			case key == indexPath:
			case pathLen >= prefixPathLen && key[:prefixPathLen] == prefixPath:
			default:
				return true
			}
			if stubCount += len(ids); stubCount > 25 {
				done = true // early out
				return false
			}
			// log.WarnF("found built quote:\nindexPath=%v\nkey=%v\nids=%v", indexPath, key, ids)
			for _, id := range ids {
				stubsLookup[id] = true
			}
		}
		return true
	})
	if done {
		return
	}

	for idx, _ := range stubsLookup {
		if idx >= 0 && idx <= f.lastStubIdx {
			shasum, _ := f.knownStubs.Load(idx)
			if stub := f.Enjin.FindPageStub(shasum); stub != nil {
				if pg, err := page.NewPageFromStub(stub, f.theme); err != nil {
					log.ErrorF("error making page from cache: %v - %v", stub.Source, err)
				} else if hash, ok := pg.Context().Get("QuoteHash").(string); ok {
					builtQuotes = append(builtQuotes, &quote.Quote{
						Url:  pg.Url(),
						Hash: hash,
					})
				} else {
					log.ErrorF("error page missing QuoteHash: %v", pg.Url())
				}
			} else {
				log.ErrorF("error finding page stub by shasum: %v", shasum)
			}
		} else {
			log.WarnF("stub index out of bounds: %v [0-%v]", idx, f.lastStubIdx)
		}
	}
	return
}