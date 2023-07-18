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

	"github.com/maruel/natural"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/golang-org-x-text/language"

	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/fs"
	"github.com/go-enjin/be/pkg/indexing"
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

const Tag feature.Tag = "pages-quote-authors"

type Feature interface {
	feature.Feature
	feature.UseMiddleware
	feature.PageTypeProcessor
	indexing.PageIndexFeature
}

type CFeature struct {
	feature.CFeature

	numQuotes int

	authorNameByAuthorKey map[string]string
	authorKeyByAuthorName map[string]string
	authorNamesByLetter   map[string][]string
}

type MakeFeature interface {
	Make() Feature
}

func New() MakeFeature {
	return NewTagged(Tag)
}

func NewTagged(tag feature.Tag) MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.FeatureTag = tag
	f.authorNameByAuthorKey = make(map[string]string)
	f.authorKeyByAuthorName = make(map[string]string)
	f.authorNamesByLetter = make(map[string][]string)
	return f
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	err = f.CFeature.Startup(ctx)
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

func (f *CFeature) AddToIndex(stub *fs.PageStub, p *page.Page) (err error) {

	if p.Type != "quote" {
		return
	}

	f.Lock()
	defer f.Unlock()

	f.numQuotes += 1

	authorKey := p.Context.String("QuoteAuthorKey", "")
	authorName := p.Context.String("QuoteAuthor", "")

	if authorKey == "" || authorName == "" {
		// bad content?
		return
	}

	if _, present := f.authorNameByAuthorKey[authorKey]; present {
		return
	}

	f.authorNameByAuthorKey[authorKey] = authorName
	f.authorKeyByAuthorName[authorName] = authorKey

	if lastName := beStrings.LastName(authorName); lastName != "" {
		fc := strings.ToLower(string(lastName[0]))
		f.authorNamesByLetter[fc] = append(f.authorNamesByLetter[fc], authorName)
	}

	return
}

func (f *CFeature) RemoveFromIndex(tag language.Tag, file string, shasum string) {
	return
}

var RxPagePath = regexp.MustCompile(`^/a/([^/]+)/??`)

func (f *CFeature) ProcessPagePath(w http.ResponseWriter, r *http.Request) (processed bool) {
	switch r.URL.Path {
	case "/a", "/a/":
		reqArgv := argv.DecodeHttpRequest(r)
		reqArgv.Path = "/authors/"
		f.Enjin.ServeRedirect(reqArgv.String(), w, r)
		processed = true
		return
	}
	if RxPagePath.MatchString(r.URL.Path) {
		m := RxPagePath.FindAllStringSubmatch(r.URL.Path, 1)
		authorKey := strings.ToLower(m[0][1])
		// log.WarnF("hit author page: %v", authorKey)

		//selectedAuthorName := f.Enjin.SelectQL(fmt.Sprintf(`SELECT DISTINCT .QuoteAuthor WITHIN (.QuoteAuthorKey == "%v")`, authorKey))
		//var authorName string
		//if name, ok := selectedAuthorName["QuoteAuthor"].(string); ok {
		//	authorName = name
		//} else {
		//	log.ErrorF("author not found by key: %v - %#+v", authorKey, selectedAuthorName)
		//	return
		//}

		var ok bool
		var authorName string
		if authorName, ok = f.authorNameByAuthorKey[authorKey]; !ok {
			return
		}

		// log.WarnF("found author: %v", authorName)

		selectedQuotes := f.Enjin.MatchQL(fmt.Sprintf(`(.QuoteAuthorKey == "%v")`, authorKey))
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

		if authorPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!a/{key}"); authorPage != nil {
			authorPage.SetSlugUrl("/a/" + authorKey)
			authorPage.Context.SetSpecific("Title", "Quoted.FYI: author "+authorName)
			authorPage.Context.SetSpecific("AuthorKey", authorKey)
			authorPage.Context.SetSpecific("AuthorName", authorName)
			authorPage.Context.SetSpecific("TotalQuotes", len(selectedQuotes))
			authorPage.Context.SetSpecific("TotalTopics", len(categoryLookup))
			authorPage.Context.SetSpecific("QuoteGroups", quoteGroups)
			authorPage.Context.SetSpecific("QuoteOtherTopics", otherTopics)
			if err := f.Enjin.ServePage(authorPage, w, r); err != nil {
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
		//results := f.Enjin.SelectQL(`SELECT DISTINCT .QuoteAuthor`)
		// log.WarnF("results: %#v", results)

		//var authors, authorLetters []string
		//var totalNumAuthors int
		//if present, ok := results["QuoteAuthor"].([]interface{}); ok {
		//	// log.WarnF("num authors: %v", len(present))
		//	for _, thing := range present {
		//		if fullName, ok := thing.(string); ok {
		//			totalNumAuthors += 1
		//			if lastName := beStrings.LastName(fullName); lastName != "" {
		//				fc := strings.ToLower(string(lastName[0]))
		//				if !beStrings.StringInSlices(fc, authorLetters) {
		//					authorLetters = append(authorLetters, fc)
		//				}
		//				if fc == groupChar {
		//					authors = append(authors, fullName)
		//				}
		//			}
		//		}
		//	}
		//}
		//sort.Sort(natural.StringSlice(authors))
		//sort.Sort(natural.StringSlice(authorLetters))

		authors := beStrings.SortedByLastName(f.authorNamesByLetter[groupChar])

		authorLetters := maps.SortedKeys(f.authorNamesByLetter)
		sort.Sort(natural.StringSlice(authorLetters))

		if listingPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!authors-key"); listingPage != nil {
			listingPage.SetSlugUrl("/authors/" + groupChar)
			listingPage.Context.SetSpecific("Authors", authors)
			listingPage.Context.SetSpecific("AuthorLetters", authorLetters)
			listingPage.Context.SetSpecific("NumAuthors", len(authors))
			listingPage.Context.SetSpecific("AuthorCharacter", groupChar)
			listingPage.Context.SetSpecific("TotalNumAuthors", len(f.authorNameByAuthorKey))
			if err := f.Enjin.ServePage(listingPage, w, r); err != nil {
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
	default:
		//pg = p
		//processed = true
		p.Context.SetSpecific("NumQuotes", f.numQuotes)
		p.Context.SetSpecific("NumAuthors", len(f.authorKeyByAuthorName))
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
	//authorLookup := make(map[string][]*quote.Quote)

	authors := p.Context.Strings("Authors")
	if len(authors) == 0 {
		//authors = maps.SortedKeysByLastName(f.authorKeyByAuthorName)
		//p.Context.SetSpecific("Authors", []interface{}{authors})
		p.Context.SetSpecific("NumAuthors", len(f.authorKeyByAuthorName))
		authorLetters := maps.SortedKeys(f.authorNamesByLetter)
		p.Context.SetSpecific("AuthorLetters", authorLetters)
		//pg = p
		//processed = true
		return
	}

	var groupIdx int
	var groupKey string
	for _, authorName := range authors {
		authorKey := quote.GetLastNameKey(authorName)
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
				Key:  f.authorKeyByAuthorName[authorName],
				Name: authorName,
			},
		)
	}

	p.Context.SetSpecific("AuthorGroups", authorGroups)

	pg = p
	processed = true
	return
}