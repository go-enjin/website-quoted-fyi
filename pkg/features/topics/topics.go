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

package topics

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

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

const Tag feature.Tag = "PagesQuoteTopics"

type Feature interface {
	feature.Feature
	feature.UseMiddleware
	feature.PageTypeProcessor
}

type CFeature struct {
	feature.CFeature
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
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	err = f.CFeature.Startup(ctx)
	return
}

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote topics middleware")
	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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

var RxPagePath = regexp.MustCompile(`^/t/([^/]+)/??`)

func (f *CFeature) ProcessPagePath(w http.ResponseWriter, r *http.Request) (processed bool) {
	switch r.URL.Path {
	case "/t", "/t/":
		reqArgv := argv.DecodeHttpRequest(r)
		reqArgv.Path = "/topics/"
		f.Enjin.ServeRedirect(reqArgv.String(), w, r)
		processed = true
		return
	}
	if RxPagePath.MatchString(r.URL.Path) {
		m := RxPagePath.FindAllStringSubmatch(r.URL.Path, 1)
		topic := strings.ToLower(m[0][1])
		topic, _ = url.PathUnescape(topic)
		topicKey := strcase.ToSnake(topic)
		// log.WarnF("hit topic page: %v", topic)

		selectedQuotes := f.Enjin.MatchQL(fmt.Sprintf(`(.QuoteCategoryKeys =~ "%v")`, topicKey))
		// log.WarnF("selected topics: %v", selectedQuotes)

		authorLookup := make(map[string][]*quote.Quote)
		for _, selectedQuote := range selectedQuotes {
			authorName, _ := selectedQuote.Context.Get("QuoteAuthor").(string)
			authorLookup[authorName] = append(authorLookup[authorName], &quote.Quote{
				Url:  selectedQuote.Url,
				Hash: selectedQuote.Context.Get("QuoteHash").(string),
			})
		}

		var topicAuthors []*quote.AuthorsGroup
		var currentAuthorGroup *quote.AuthorsGroup
		for _, authorName := range maps.SortedKeysByLastName(authorLookup) {
			groupKey := quote.GetLastNameCharacter(authorName)
			if currentAuthorGroup == nil {
				currentAuthorGroup = &quote.AuthorsGroup{
					Key: groupKey,
				}
			} else if groupKey != currentAuthorGroup.Key {
				topicAuthors = append(topicAuthors, currentAuthorGroup)
				currentAuthorGroup = &quote.AuthorsGroup{
					Key: groupKey,
				}
			}
			authorKey := strcase.ToSnake(authorName)
			currentAuthorGroup.Authors = append(currentAuthorGroup.Authors, &quote.Author{
				Url:    "/a/" + authorKey,
				Key:    authorKey,
				Name:   authorName,
				Quotes: authorLookup[authorName],
			})
		}
		if currentAuthorGroup != nil {
			topicAuthors = append(topicAuthors, currentAuthorGroup)
			currentAuthorGroup = nil
		}

		if topicPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!t/{key}"); topicPage != nil {
			topicPage.SetSlugUrl("/t/" + topic)
			topicPage.Context.SetSpecific("Title", "Quoted.FYI: topic "+topic)
			topicPage.Context.SetSpecific("Topic", topic)
			topicPage.Context.SetSpecific("TotalQuotes", len(selectedQuotes))
			topicPage.Context.SetSpecific("TotalAuthors", len(authorLookup))
			topicPage.Context.SetSpecific("TopicAuthors", topicAuthors)
			if err := f.Enjin.ServePage(topicPage, w, r); err != nil {
				log.ErrorF("error serving topics listing page: %v", err)
			} else {
				processed = true
			}
		}
	}
	return
}

var RxGroupPath = regexp.MustCompile(`^/topics/([a-zA-Z0-9])?/??`)

func (f *CFeature) ProcessGroupPath(w http.ResponseWriter, r *http.Request) (processed bool) {
	if RxGroupPath.MatchString(r.URL.Path) {
		m := RxGroupPath.FindAllStringSubmatch(r.URL.Path, 1)
		groupChar := strings.ToLower(m[0][1])
		// log.WarnF("hit topics group: %v", groupChar)
		// results := f.Enjin.SelectQL(`SELECT DISTINCT .QuoteCategories`)
		results := f.Enjin.SelectQL(`SELECT DISTINCT .QuoteCategories`)
		// log.WarnF("results: %#v", results)
		var topics, topicLetters []string
		var totalNumTopics int
		if present, ok := results["QuoteCategories"].([]interface{}); ok {
			// log.WarnF("num topics: %v", len(present))
			for _, thing := range present {
				if topic, ok := thing.(string); ok {
					totalNumTopics += 1
					if topic == "" {
						continue
					}
					fc := strings.ToLower(string(topic[0]))
					if !beStrings.StringInSlices(fc, topicLetters) {
						topicLetters = append(topicLetters, fc)
					}
					if fc == groupChar {
						topics = append(topics, topic)
					}
				}
			}
		}

		sort.Sort(natural.StringSlice(topics))
		sort.Sort(natural.StringSlice(topicLetters))

		topicLookup := make(map[string]*quote.TopicAuthorsGroup)
		for _, topic := range topics {
			key := quote.GetFirstCharacters(3, topic)
			if _, exists := topicLookup[key]; !exists {
				topicLookup[key] = &quote.TopicAuthorsGroup{
					Key: key,
				}
			}
			topicLookup[key].Topics = append(topicLookup[key].Topics, &quote.TopicAuthors{
				Key:  strcase.ToSnake(topic),
				Name: topic,
			})
		}

		topicGroups := make([]*quote.TopicAuthorsGroup, 0)
		for _, key := range maps.SortedKeys(topicLookup) {
			topicGroups = append(topicGroups, topicLookup[key])
		}

		if listingPage := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), "!topics/{key}"); listingPage != nil {
			listingPage.SetSlugUrl("/topics/" + groupChar)
			listingPage.Context.SetSpecific("Topics", topics)
			listingPage.Context.SetSpecific("TopicLetters", topicLetters)
			listingPage.Context.SetSpecific("NumTopics", len(topics))
			listingPage.Context.SetSpecific("TopicGroups", topicGroups)
			listingPage.Context.SetSpecific("TopicCharacter", groupChar)
			listingPage.Context.SetSpecific("TotalNumTopics", totalNumTopics)
			if err := f.Enjin.ServePage(listingPage, w, r); err != nil {
				log.ErrorF("error serving topics listing page: %v", err)
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
	case "topics":
		pg, redirect, processed, err = f.ProcessGroupsPageType(r, p)
	case "topic":
		pg, redirect, processed, err = f.ProcessSinglePageType(r, p)
	}

	return
}

func (f *CFeature) ProcessSinglePageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit topic page type: %v", p.Url)
	return
}

func (f *CFeature) ProcessGroupsPageType(r *http.Request, p *page.Page) (pg *page.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit topic group type: %v", p.Url)
	pg = p
	processed = true
	return
}