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

package words

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-corelibs/maps"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/request/argv"
	"github.com/go-enjin/be/types/page"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) ProcessPagePath(pathWord string, w http.ResponseWriter, r *http.Request) {

	var wordPage feature.Page
	if wordPage = f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!w/{key}"); wordPage == nil {
		log.ErrorRF(r, "error words page not found: !w/{key}")
		f.Enjin.ServeInternalServerError(w, r)
		return
	}

	reqArgv := argv.Get(r)

	numPerPage, pageNumber := wordPage.Context().Int("NumPerPage", 100), 0
	if reqArgv.NumPerPage > 0 {
		numPerPage = reqArgv.NumPerPage
	}
	if reqArgv.PageNumber > 0 {
		pageNumber = reqArgv.PageNumber
	}

	var word string
	if clean, err := url.PathUnescape(pathWord); err != nil {
		log.ErrorF("error unescaping url path: %v - %v", pathWord, err)
		f.Enjin.ServeNotFound(w, r)
		return
	} else {
		word = clean
	}
	word = strings.ToLower(word)
	if word != "" && word != pathWord {
		ra := reqArgv.Copy()
		ra.Path = "/w/" + word
		if pageNumber > 0 {
			ra.PageNumber = pageNumber
			ra.NumPerPage = numPerPage
		} else {
			ra.PageNumber = -1
			ra.NumPerPage = -1
		}
		// log.WarnF("redirecting to clean keyword: %v -> %v from: %v", pathWord, ra.String(), reqArgv.String())
		f.Enjin.ServeRedirect(ra.String(), w, r)
		return
	}

	if wid, ok := f.dbh.GetWidFrom(word); !ok {
		f.Enjin.ServeNotFound(w, r)
		return
	} else {
		word, _ = f.dbh.GetWordFrom(wid)
	}

	// log.WarnF("hit word page: %v", word)

	selectedStubs, totalStubsCount := f.dbh.GetPaginatedWordPageStubs(word, pageNumber, numPerPage)
	selectedStubsCount := len(selectedStubs)
	//log.WarnF("found %d stubs for word: %v (paginating %d)", totalStubsCount, word, selectedStubsCount)

	totalNumPages := int(float64(selectedStubsCount) / float64(numPerPage))
	matchingStubs := selectedStubs
	if selectedStubsCount%numPerPage != 0 {
		totalNumPages += 1 // extra page for the remainder
	}
	if totalNumPages == 0 {
		f.Enjin.ServeNotFound(w, r)
		return
	}
	if pageNumber >= totalNumPages {
		reqArgv.PageNumber = totalNumPages - 1
		reqArgv.NumPerPage = numPerPage
		// log.WarnF("redirecting to last page, page number too large: %v (%v) - %v", pageNumber, totalNumPages, reqArgv.String())
		f.Enjin.ServeRedirect(reqArgv.String(), w, r)
		return
	}

	var selectedQuotes []feature.Page
	topicsLookup := make(map[string][]*quote.Quote)
	ectx := f.Enjin.Context(r)
	for _, stub := range matchingStubs {
		if pg, err := page.NewPageFromStub(stub, f.theme, ectx); err != nil {
			log.ErrorF("error making page from cache: %v", err)
		} else {
			selectedQuotes = append(selectedQuotes, pg)
			if topics, ok := pg.Context().Get("QuoteCategories").([]string); ok {
				for _, topic := range topics {
					if topic == "" {
						continue
					}
					topicsLookup[topic] = append(topicsLookup[topic], &quote.Quote{
						Url:  pg.Url(),
						Hash: pg.Context().Get("QuoteHash").(string),
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
			Key:    quote.FlattenContent(topic),
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
	wordPage.Context().SetSpecific("Title", `Quoted.FYI: word "`+word+`"`)
	wordPage.Context().SetSpecific("Word", word)
	wordPage.Context().SetSpecific("TotalQuotes", totalStubsCount)
	wordPage.Context().SetSpecific("TotalNumPages", totalNumPages)
	wordPage.Context().SetSpecific("PageNumber", pageNumber)
	wordPage.Context().SetSpecific("NumPerPage", numPerPage)
	wordPage.Context().SetSpecific("TotalTopics", len(topicsLookup))
	wordPage.Context().SetSpecific("WordGroups", wordGroups)
	if err := f.Enjin.ServePage(wordPage, w, r); err != nil {
		log.ErrorF("error serving words listing page: %v", err)
	}
	return
}
