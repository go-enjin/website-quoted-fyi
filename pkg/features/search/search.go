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

package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/maps"
	"github.com/go-enjin/be/pkg/page"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "search-quoted"

type Feature interface {
	feature.Feature
	search.ResultsPostProcessor
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.FeatureTag = Tag
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	err = f.CFeature.Startup(ctx)
	return
}

func (f *CFeature) SearchResultsPostProcess(p *page.Page) {
	var query string
	if query = p.Context.String("SiteSearchQuery", ""); query == "" {
		p.Title = "Quoted.FYI: Search"
	} else {
		p.Title = "Quoted.FYI: Searching"
	}
	p.Context.SetSpecific("Title", p.Title)

	langTag := f.Enjin.SiteDefaultLanguage()
	if results, ok := p.Context.Get("SiteSearchResults").(*bleve.SearchResult); ok {

		authorLookup := make(map[string][]*quote.Quote)

		// found := make(map[string]*page.Page)
		for _, hit := range results.Hits {
			if pg := f.Enjin.FindPage(langTag, hit.ID); pg != nil {
				if pg.Type != "quote" {
					continue
				}
				// found[pg.Url] = pg
				qAuthor := pg.Context.Get("QuoteAuthor").(string)
				authorLookup[qAuthor] = append(authorLookup[qAuthor], &quote.Quote{
					Url:  pg.Url,
					Hash: pg.Context.Get("QuoteHash").(string),
				})
			} else {
				log.ErrorF("page not found: %v", hit.ID)
			}
		}

		authorNames := maps.SortedKeysByLastName(authorLookup)
		quotedGroups := make([]*quote.AuthorsGroup, 0)

		var groupIdx int
		var groupKey string
		for _, authorName := range authorNames {
			authorKey := quote.GetLastNameCharacter(authorName)
			if groupKey != authorKey {
				quotedGroups = append(quotedGroups, &quote.AuthorsGroup{
					Key: authorKey,
				})
				groupIdx = len(quotedGroups) - 1
				groupKey = authorKey
			}
			quotedGroups[groupIdx].Authors = append(
				quotedGroups[groupIdx].Authors,
				&quote.Author{
					Key:    strcase.ToSnake(authorName),
					Name:   authorName,
					Quotes: authorLookup[authorName],
				},
			)
		}

		p.Context.SetSpecific("QuotedGroups", quotedGroups)

	}
}