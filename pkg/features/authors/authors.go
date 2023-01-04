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

package authors

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	"github.com/maruel/natural"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
	"github.com/go-enjin/be/pkg/page"
	"github.com/go-enjin/be/pkg/request/argv"
	beStrings "github.com/go-enjin/be/pkg/strings"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "PagesQuoteAuthors"

type Feature interface {
	feature.Middleware
	feature.PageTypeProcessor
}

type CFeature struct {
	feature.CMiddleware

	cli   *cli.Context
	enjin feature.Internals

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
}

func (f *CFeature) Tag() (tag feature.Tag) {
	tag = Tag
	return
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.enjin = enjin
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	f.cli = ctx
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote authors middleware")
	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// TODO: redirect /a/ -> /authors/, etc

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

var RxPagePath = regexp.MustCompile(`^/a/([^/]+)/??`)

func (f *CFeature) ProcessPagePath(w http.ResponseWriter, r *http.Request) (processed bool) {
	switch r.URL.Path {
	case "/a", "/a/":
		reqArgv := argv.DecodeHttpRequest(r)
		reqArgv.Path = "/authors/"
		f.enjin.ServeRedirect(reqArgv.String(), w, r)
		processed = true
		return
	}
	if RxPagePath.MatchString(r.URL.Path) {
		m := RxPagePath.FindAllStringSubmatch(r.URL.Path, 1)
		authorKey := strings.ToLower(m[0][1])
		// log.WarnF("hit author page: %v", authorKey)

		selectedAuthorName := f.enjin.SelectQL(fmt.Sprintf(`SELECT DISTINCT .QuoteAuthor WITHIN (.QuoteAuthorKey == "%v")`, authorKey))
		var authorName string
		if name, ok := selectedAuthorName["QuoteAuthor"].(string); ok {
			authorName = name
		} else {
			log.ErrorF("author not found by key: %v", authorKey)
			return
		}
		// log.WarnF("found author: %v", authorName)

		selectedQuotes := f.enjin.MatchQL(fmt.Sprintf(`(.QuoteAuthorKey == "%v")`, authorKey))
		categoryLookup := make(map[string][]*quote.Quote)
		for _, selectedQuote := range selectedQuotes {
			if categories, ok := selectedQuote.Context.Get("QuoteCategories").([]string); ok {
				for _, category := range categories {
					found := false
					for _, categoryQuote := range categoryLookup[category] {
						if found = categoryQuote.Url == selectedQuote.Url; found {
							break
						}
					}
					if !found {
						categoryLookup[category] = append(categoryLookup[category], &quote.Quote{
							Url:  selectedQuote.Url,
							Hash: selectedQuote.Context.Get("QuoteHash").(string),
						})
					}
				}
			}
		}

		quoteGroups := make([]*quote.QuotesGroups, 0)
		otherTopics := make([]*quote.Quote, 0)

		var currentGroups *quote.QuotesGroups
		for _, categoryKey := range maps.SortedKeys(categoryLookup) {
			if len(categoryLookup[categoryKey]) == 1 {
				singleQuote := categoryLookup[categoryKey][0]
				found := false
				for _, categoryQuote := range otherTopics {
					if found = categoryQuote.Url == singleQuote.Url; found {
						break
					}
				}
				if !found {
					otherTopics = append(otherTopics, singleQuote)
				}
				continue
			}
			groupsKey := string(categoryKey[0])
			if currentGroups == nil {
				currentGroups = &quote.QuotesGroups{
					Key: groupsKey,
				}
			} else if currentGroups.Key != groupsKey {
				quoteGroups = append(quoteGroups, currentGroups)
				currentGroups = &quote.QuotesGroups{
					Key: groupsKey,
				}
			}
			currentGroups.Groups = append(currentGroups.Groups, &quote.QuotesGroup{
				Key:    categoryKey,
				Quotes: categoryLookup[categoryKey],
			})
		}
		if currentGroups != nil {
			quoteGroups = append(quoteGroups, currentGroups)
		}

		if authorPage := f.enjin.FindPage(f.enjin.SiteDefaultLanguage(), "!a/{key}"); authorPage != nil {
			authorPage.SetSlugUrl("/a/" + authorKey)
			authorPage.Context.SetSpecific("Title", "Quoted.FYI: author "+authorName)
			authorPage.Context.SetSpecific("AuthorKey", authorKey)
			authorPage.Context.SetSpecific("AuthorName", authorName)
			authorPage.Context.SetSpecific("TotalQuotes", len(selectedQuotes))
			authorPage.Context.SetSpecific("TotalTopics", len(categoryLookup))
			authorPage.Context.SetSpecific("QuoteGroups", quoteGroups)
			authorPage.Context.SetSpecific("QuoteOtherTopics", otherTopics)
			if err := f.enjin.ServePage(authorPage, w, r); err != nil {
				log.ErrorF("error serving authors listing page: %v", err)
			} else {
				processed = true
			}
		}
	}
	return
}

var RxGroupPath = regexp.MustCompile(`^/authors/([a-z])?/??`)

func (f *CFeature) ProcessGroupPath(w http.ResponseWriter, r *http.Request) (processed bool) {
	if RxGroupPath.MatchString(r.URL.Path) {
		m := RxGroupPath.FindAllStringSubmatch(r.URL.Path, 1)
		groupChar := strings.ToLower(m[0][1])
		// log.WarnF("hit authors group: %v", groupChar)
		results := f.enjin.SelectQL(`SELECT DISTINCT .QuoteAuthor`)
		// log.WarnF("results: %#v", results)

		var authors, authorLetters []string
		var totalNumAuthors int
		if present, ok := results["QuoteAuthor"].([]interface{}); ok {
			// log.WarnF("num authors: %v", len(present))
			for _, thing := range present {
				if fullName, ok := thing.(string); ok {
					totalNumAuthors += 1
					if lastName := beStrings.LastName(fullName); lastName != "" {
						fc := strings.ToLower(string(lastName[0]))
						if !beStrings.StringInSlices(fc, authorLetters) {
							authorLetters = append(authorLetters, fc)
						}
						if fc == groupChar {
							authors = append(authors, fullName)
						}
					}
				}
			}
		}

		sort.Sort(natural.StringSlice(authors))
		sort.Sort(natural.StringSlice(authorLetters))

		if listingPage := f.enjin.FindPage(f.enjin.SiteDefaultLanguage(), "!authors-key"); listingPage != nil {
			listingPage.SetSlugUrl("/authors/" + groupChar)
			listingPage.Context.SetSpecific("Authors", authors)
			listingPage.Context.SetSpecific("AuthorLetters", authorLetters)
			listingPage.Context.SetSpecific("NumAuthors", len(authors))
			listingPage.Context.SetSpecific("AuthorCharacter", groupChar)
			listingPage.Context.SetSpecific("TotalNumAuthors", totalNumAuthors)
			if err := f.enjin.ServePage(listingPage, w, r); err != nil {
				log.ErrorF("error serving authors listing page: %v", err)
			} else {
				processed = true
			}
		} else {
			log.ErrorF("error finding authors key page: %v", groupChar)
		}
	}
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// reqArgv := site.GetRequestArgv(r)

	switch p.Type {
	case "authors":
		pg, redirect, processed, err = f.ProcessGroupsPageType(r, p)
	case "author":
		pg, redirect, processed, err = f.ProcessSinglePageType(r, p)
	}

	return
}

func (f *CFeature) ProcessSinglePageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {

	// log.WarnF("hit author page type: %v", p.Url)
	// quoteGroups := make([]*quote.QuotesGroup, 0)
	// quoteLookup := make(map[string][]*quote.Quote)

	return
}

func (f *CFeature) ProcessGroupsPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	authorGroups := make([]*quote.AuthorsGroup, 0)
	authorLookup := make(map[string][]*quote.Quote)

	authors := p.Context.Strings("Authors")
	selected := f.enjin.SelectQL(`SELECT .Url, .QuoteHash, .QuoteAuthor`)
	selectedUrls, _ := maps.TransformAnyToStringSlice(selected["Url"])
	selectedHashes, _ := maps.TransformAnyToStringSlice(selected["QuoteHash"])
	selectedAuthors, _ := maps.TransformAnyToStringSlice(selected["QuoteAuthor"])
	for idx := 0; idx < len(selectedAuthors); idx++ {
		author := selectedAuthors[idx]
		if beStrings.StringInSlices(author, authors) {
			authorLookup[author] = append(authorLookup[author], &quote.Quote{
				Url:  selectedUrls[idx],
				Hash: selectedHashes[idx],
			})
		}
	}

	var groupIdx int
	var groupKey string
	for _, author := range maps.SortedKeysByLastName(authorLookup) {
		authorKey := quote.GetLastNameKey(author)
		if len(authorGroups) == 0 || groupKey != authorKey {
			authorGroups = append(authorGroups, &quote.AuthorsGroup{
				Key: authorKey,
			})
			groupIdx = len(authorGroups) - 1
			groupKey = authorKey
		}
		authorGroups[groupIdx].Authors = append(
			authorGroups[groupIdx].Authors,
			&quote.Author{
				Key:    strcase.ToSnake(author),
				Name:   author,
				Quotes: authorLookup[author],
			},
		)
	}

	p.Context.SetSpecific("AuthorGroups", authorGroups)

	pg = p
	processed = true
	return
}