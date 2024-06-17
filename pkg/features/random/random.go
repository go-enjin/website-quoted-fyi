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

package random

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/dbh"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "random-pages"

type Feature interface {
	feature.Feature
	feature.PageTypeProcessor
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature

	dbh dbh.Feature
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = Tag
	f.CFeature.Construct(f)
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

func (f *CFeature) PostStartup(ctx *cli.Context) (err error) {
	if tfs := feature.FilterTyped[dbh.Feature](f.Enjin.Features().List()); len(tfs) > 0 {
		f.dbh = tfs[0]
	} else {
		err = fmt.Errorf("a dbh.Feature is required")
		return
	}
	return
}

func (f *CFeature) PageTypeNames() (names []string) {
	names = append(names, "random")
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	if p.Type() == "random" {

		if v, ok := p.Context().Get("Random").(string); !ok {
			log.ErrorRF(r, "random page without random key: %v", p.Url())
			redirect = "/random"
			return
		} else {
			switch v {

			case "a", "author":
				author := f.dbh.GetRandomAuthor()
				authorKey := quote.FlattenContent(author)
				p.Context().SetSpecific("AuthorKey", authorKey)
				p.Context().SetSpecific("AuthorName", author)
				p.Context().SetSpecific("MetaRefresh", "5; url=/a/"+url.PathEscape(authorKey))

			case "t", "topic":
				topic := f.dbh.GetRandomTopic()
				p.Context().SetSpecific("Topic", topic)
				p.Context().SetSpecific("MetaRefresh", "5; url=/t/"+url.PathEscape(topic))

			case "q", "quote":
				quoteUrl, quoteHash := f.dbh.GetRandomQuoteUrl()
				p.Context().SetSpecific("QuoteUrl", quoteUrl)
				p.Context().SetSpecific("QuoteHash", quoteHash)
				p.Context().SetSpecific("MetaRefresh", "5; url="+quoteUrl)

			case "w", "word":
				word := f.dbh.GetRandomWord()
				p.Context().SetSpecific("Word", word)
				p.Context().SetSpecific("MetaRefresh", "5; url=/w/"+url.PathEscape(word))

			default:
				log.ErrorRF(r, "random page with invalid random key: %v", v)
				redirect = "/random"
				return
			}
		}

		pg = p
		processed = true
	}
	return
}
