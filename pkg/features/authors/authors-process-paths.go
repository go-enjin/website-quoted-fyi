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

package authors

import (
	"net/http"
	"sort"

	"github.com/maruel/natural"

	"github.com/go-corelibs/maps"
	"github.com/go-corelibs/strings"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/types/page"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) ProcessPagePath(authorKey string, w http.ResponseWriter, r *http.Request) {

	// log.WarnF("hit author page: %v", authorKey)

	var fullName string
	if fn, _, k, ok := f.dbh.GetAuthorNames(authorKey); ok {
		fullName = fn
		authorKey = k
	}

	t := f.Enjin.MustGetTheme()
	ectx := f.Enjin.Context(r)

	var selectedQuotes []feature.Page

	if results, err := f.eql.PerformQuery(
		`QUERY WITHIN (author.Flat == %q) OR (author.FullName == %q)`,
		authorKey, authorKey,
	); err != nil {
		log.ErrorRF(r, "error finding quotes for author %q: %v", authorKey, err)
		f.Enjin.ServeNotFound(w, r)
		return
	} else {
		for _, stub := range results {
			if p, ee := page.NewPageFromStub(stub, t, ectx); ee == nil {
				selectedQuotes = append(selectedQuotes, p)
			}
		}
	}

	categoryLookup := make(map[string][]*quote.Quote)
	for _, selectedQuote := range selectedQuotes {
		if categories, ok := selectedQuote.Context().Get("QuoteCategories").([]string); ok {
			for _, category := range categories {
				found := false
				for _, categoryQuote := range categoryLookup[category] {
					if found = categoryQuote.Url == selectedQuote.Url(); found {
						break
					}
				}
				if !found {
					categoryLookup[category] = append(categoryLookup[category], &quote.Quote{
						Url:  selectedQuote.Url(),
						Hash: selectedQuote.Context().Get("QuoteHash").(string),
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

	if authorPage := f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!a/{key}"); authorPage != nil {
		authorPage.SetSlugUrl("/a/" + authorKey)
		authorPage.Context().SetSpecific("Title", "Quoted.FYI: author "+fullName)
		authorPage.Context().SetSpecific("AuthorKey", authorKey)
		authorPage.Context().SetSpecific("AuthorName", fullName)
		authorPage.Context().SetSpecific("TotalQuotes", len(selectedQuotes))
		authorPage.Context().SetSpecific("TotalTopics", len(categoryLookup))
		authorPage.Context().SetSpecific("QuoteGroups", quoteGroups)
		authorPage.Context().SetSpecific("QuoteOtherTopics", otherTopics)
		if err := f.Enjin.ServePage(authorPage, w, r); err != nil {
			log.ErrorF("error serving authors listing page: %v", err)
		}
	} else {
		log.ErrorRF(r, "error authors page not found: !a/{key}")
		f.Enjin.ServeInternalServerError(w, r)
	}
	return
}

func (f *CFeature) ProcessGroupPath(groupChar string, w http.ResponseWriter, r *http.Request) {
	// log.WarnF("hit authors group: %v", groupChar)

	var authorNames []string

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP author.FullName WITHIN author.Letter == %q`,
		groupChar,
	); err != nil {
		log.ErrorRF(r, "error getting authors for group %q: %v", groupChar, err)
		f.Enjin.ServeNotFound(w, r)
		return
	} else {
		authorNames = results.StringValues("FullName")
	}

	sort.Sort(natural.StringSlice(authorNames))
	authors := strings.SortedByLastName(authorNames)

	if listingPage := f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!authors-key"); listingPage != nil {
		listingPage.SetSlugUrl("/authors/" + groupChar)
		listingPage.Context().SetSpecific("Authors", authors)
		listingPage.Context().SetSpecific("AuthorLetters", f.authorLetters)
		listingPage.Context().SetSpecific("TotalNumAuthors", len(authors))
		listingPage.Context().SetSpecific("AuthorCharacter", groupChar)
		if err := f.Enjin.ServePage(listingPage, w, r); err != nil {
			log.ErrorF("error serving authors listing page: %v", err)
		}
	} else {
		log.ErrorRF(r, "error authors page not found: !authors-key")
		f.Enjin.ServeInternalServerError(w, r)
	}
	return
}
