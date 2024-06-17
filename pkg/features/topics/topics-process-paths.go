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

package topics

import (
	"net/http"

	"github.com/go-corelibs/maps"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/types/page"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) ProcessPagePath(topic string, w http.ResponseWriter, r *http.Request) {
	var ok bool
	var flat, name string
	if name, flat, ok = f.dbh.GetTopicNames(topic); ok {
		topic = name
	} else {
		f.Enjin.ServeNotFound(w, r)
		return
	}

	t := f.Enjin.MustGetTheme()
	ectx := f.Enjin.Context(r)
	var selectedQuotes []feature.Page

	if results, err := f.eql.PerformQuery(
		`QUERY WITHIN topic.Flat == %q`,
		flat,
	); err != nil {
		log.ErrorRF(r, "error finding quotes for topic %q: %v", flat, err)
		f.Enjin.ServeNotFound(w, r)
		return
	} else {
		for _, stub := range results {
			if p, ee := page.NewPageFromStub(stub, t, ectx); ee == nil {
				selectedQuotes = append(selectedQuotes, p)
			}
		}
	}

	// log.WarnF("selected topics: %v", selectedQuotes)

	authorLookup := make(map[string][]*quote.Quote)
	for _, selectedQuote := range selectedQuotes {
		authorName, _ := selectedQuote.Context().Get("QuoteAuthor").(string)
		authorLookup[authorName] = append(authorLookup[authorName], &quote.Quote{
			Url:  selectedQuote.Url(),
			Hash: selectedQuote.Context().Get("QuoteHash").(string),
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
		authorKey := quote.FlattenContent(authorName)
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

	if topicPage := f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!t/{key}"); topicPage != nil {
		topicPage.SetSlugUrl("/t/" + topic)
		topicPage.Context().SetSpecific("Title", "Quoted.FYI: topic "+topic)
		topicPage.Context().SetSpecific("Topic", topic)
		topicPage.Context().SetSpecific("TotalQuotes", len(selectedQuotes))
		topicPage.Context().SetSpecific("TotalAuthors", len(authorLookup))
		topicPage.Context().SetSpecific("TopicAuthors", topicAuthors)
		if err := f.Enjin.ServePage(topicPage, w, r); err != nil {
			log.ErrorRF(r, "error serving topics listing page: %v", err)
		}
	} else {
		log.ErrorRF(r, "error topics page not found: !t/{key}")
		f.Enjin.ServeInternalServerError(w, r)
	}
	return
}

func (f *CFeature) ProcessGroupPath(groupChar string, w http.ResponseWriter, r *http.Request) {
	var topics []string

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP topic.Topic WITHIN topic.Letter == %q`,
		groupChar,
	); err != nil {
		log.ErrorRF(r, "error getting topics for group %q: %v", groupChar, err)
		f.Enjin.ServeNotFound(w, r)
		return
	} else {
		topics = results.StringValues("topic")
	}

	topicLookup := make(map[string]*quote.TopicAuthorsGroup)
	for _, topic := range topics {
		key := quote.GetFirstCharacters(3, topic)
		if _, exists := topicLookup[key]; !exists {
			topicLookup[key] = &quote.TopicAuthorsGroup{
				Key: key,
			}
		}
		topicLookup[key].Topics = append(topicLookup[key].Topics, &quote.TopicAuthors{
			Key:  quote.FlattenContent(topic),
			Name: topic,
		})
	}

	topicGroups := make([]*quote.TopicAuthorsGroup, 0)
	for _, key := range maps.SortedKeys(topicLookup) {
		topicGroups = append(topicGroups, topicLookup[key])
	}

	if listingPage := f.Enjin.FindPage(r, f.Enjin.SiteDefaultLanguage(), "!topics/{key}"); listingPage != nil {
		listingPage.SetSlugUrl("/topics/" + groupChar)
		listingPage.Context().SetSpecific("Topics", topics)
		listingPage.Context().SetSpecific("TopicLetters", f.topicLetters)
		listingPage.Context().SetSpecific("TotalNumTopics", len(topics))
		listingPage.Context().SetSpecific("TopicGroups", topicGroups)
		listingPage.Context().SetSpecific("TopicCharacter", groupChar)
		if err := f.Enjin.ServePage(listingPage, w, r); err != nil {
			log.ErrorRF(r, "error serving topics listing page: %v", err)
		}
	} else {
		log.ErrorRF(r, "error topics page not found: !topics/{key}")
		f.Enjin.ServeInternalServerError(w, r)
	}
	return
}
