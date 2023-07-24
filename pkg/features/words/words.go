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

package words

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/fs"
	"github.com/go-enjin/be/pkg/indexing"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
	"github.com/go-enjin/be/pkg/page"
	"github.com/go-enjin/be/pkg/request/argv"
	"github.com/go-enjin/be/pkg/theme"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "quote-word-pages"

type Feature interface {
	feature.Feature
	feature.UseMiddleware
	feature.PageTypeProcessor
}

type CFeature struct {
	feature.CFeature

	kwp   indexing.KeywordProvider
	theme *theme.Theme

	sync.RWMutex
}

type MakeFeature interface {
	Make() Feature
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.FeatureTag = Tag
	return f
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
	f.kwp = nil
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
	if t, err := f.Enjin.GetTheme(); err != nil {
		log.FatalF("error getting enjin theme: %v", err)
	} else {
		f.theme = t
	}
	for _, feat := range f.Enjin.Features() {
		if kwp, ok := feat.(indexing.KeywordProvider); ok {
			f.kwp = kwp
			break
		}
	}
	if f.kwp == nil {
		log.FatalF("%v requires a pagecache.KeywordProvider feature to be present", f.Tag())
	}
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	err = f.CFeature.Startup(ctx)
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote words middleware")
	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			switch {
			case f.ProcessPagePath(w, r):
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

var RxPagePath = regexp.MustCompile(`^/w/([^/]+)/??`)

func (f *CFeature) ProcessPagePath(w http.ResponseWriter, r *http.Request) (processed bool) {
	if RxPagePath.MatchString(r.URL.Path) {
		if wordPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!w/{key}"); wordPage != nil {

			m := RxPagePath.FindAllStringSubmatch(r.URL.Path, 1)

			reqArgv := argv.DecodeHttpRequest(r)

			numPerPage, pageNumber := wordPage.Context.Int("NumPerPage", 100), 0
			if reqArgv.NumPerPage > 0 {
				numPerPage = reqArgv.NumPerPage
			}
			if reqArgv.PageNumber > 0 {
				pageNumber = reqArgv.PageNumber
			}

			var word string
			if clean, err := url.PathUnescape(m[0][1]); err != nil {
				log.ErrorF("error unescaping url path: %v - %v", m[0][1], err)
				f.Enjin.ServeNotFound(w, r)
				processed = true
				return
			} else {
				word = clean
			}
			word = strings.ToLower(word)
			if word != "" && word != m[0][1] {
				ra := reqArgv.Copy()
				ra.Path = "/w/" + word
				if pageNumber > 0 {
					ra.PageNumber = pageNumber
					ra.NumPerPage = numPerPage
				} else {
					ra.PageNumber = -1
					ra.NumPerPage = -1
				}
				// log.WarnF("redirecting to clean keyword: %v -> %v from: %v", m[0][1], ra.String(), reqArgv.String())
				f.Enjin.ServeRedirect(ra.String(), w, r)
				processed = true
				return
			}
			// log.WarnF("hit word page: %v", word)

			selectedStubs := f.kwp.KeywordStubs(word)
			selectedStubsCount := len(selectedStubs)
			log.WarnF("found %d stubs for word: %v", selectedStubsCount, word)

			var totalNumPages int
			var matchingStubs []*fs.PageStub

			if selectedStubsCount <= numPerPage {
				totalNumPages = 1
				matchingStubs = selectedStubs
			} else {
				totalNumPages = int(float64(selectedStubsCount) / float64(numPerPage))
				if selectedStubsCount%numPerPage != 0 {
					totalNumPages += 1
				}
				if totalNumPages == 0 {
					f.Enjin.Serve404(w, r)
					processed = true
					return
				}
				if pageNumber >= totalNumPages {
					reqArgv.PageNumber = totalNumPages - 1
					reqArgv.NumPerPage = numPerPage
					// log.WarnF("redirecting to last page, page number too large: %v (%v) - %v", pageNumber, totalNumPages, reqArgv.String())
					f.Enjin.ServeRedirect(reqArgv.String(), w, r)
					processed = true
					return
				}

				startIndex := pageNumber * numPerPage
				endIndex := startIndex + numPerPage
				if endIndex >= selectedStubsCount {
					endIndex = selectedStubsCount - 1
				}
				matchingStubs = selectedStubs[startIndex:endIndex]
			}

			var selectedQuotes []*page.Page
			topicsLookup := make(map[string][]*quote.Quote)
			for _, stub := range matchingStubs {
				if pg, err := page.NewFromPageStub(stub, f.theme); err != nil {
					log.ErrorF("error making page from cache: %v", err)
				} else {
					selectedQuotes = append(selectedQuotes, pg)
					if topics, ok := pg.Context.Get("QuoteCategories").([]string); ok {
						for _, topic := range topics {
							if topic == "" {
								continue
							}
							topicsLookup[topic] = append(topicsLookup[topic], &quote.Quote{
								Url:  pg.Url,
								Hash: pg.Context.Get("QuoteHash").(string),
							})
						}
					}
				}
			}

			wordsLookup := make(map[string]*quote.WordTopicGroup)
			for _, topic := range maps.SortedKeys(topicsLookup) {
				firstLetter := strings.ToLower(string(topic[0]))
				if _, exists := wordsLookup[firstLetter]; !exists {
					wordsLookup[firstLetter] = &quote.WordTopicGroup{
						Key: firstLetter,
					}
				}
				wordsLookup[firstLetter].Topics = append(wordsLookup[firstLetter].Topics, &quote.TopicQuotes{
					Key:    strcase.ToSnake(topic),
					Name:   topic,
					Quotes: topicsLookup[topic],
				})
			}

			wordGroups := make([]*quote.WordTopicGroup, 0)
			for _, key := range maps.SortedKeys(wordsLookup) {
				wordGroups = append(wordGroups, wordsLookup[key])
			}

			// log.WarnF("selected %d quotes, npp=%v, pn=%v", len(selectedQuotes), numPerPage, pageNumber)

			wordPage.SetSlugUrl("/w/" + word)
			wordPage.Context.SetSpecific("Title", `Quoted.FYI: word "`+word+`"`)
			wordPage.Context.SetSpecific("Word", word)
			wordPage.Context.SetSpecific("TotalQuotes", selectedStubsCount)
			wordPage.Context.SetSpecific("TotalNumPages", totalNumPages)
			wordPage.Context.SetSpecific("PageNumber", pageNumber)
			wordPage.Context.SetSpecific("NumPerPage", numPerPage)
			wordPage.Context.SetSpecific("TotalTopics", len(topicsLookup))
			wordPage.Context.SetSpecific("WordGroups", wordGroups)
			if err := f.Enjin.ServePage(wordPage, w, r); err != nil {
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
	case "words":
		pg, redirect, processed, err = f.ProcessGroupsPageType(r, p)
	case "word":
		pg, redirect, processed, err = f.ProcessSinglePageType(r, p)
	}

	return
}

func (f *CFeature) ProcessSinglePageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit word page type: %v", p.Url)
	return
}

func (f *CFeature) ProcessGroupsPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit words group type: %v", p.Url)
	return
}