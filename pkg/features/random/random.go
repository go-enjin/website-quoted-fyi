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

package random

import (
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
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

	SetKeywordProvider(kwp feature.Tag) MakeFeature
}

type CFeature struct {
	feature.CFeature

	kwpTag feature.Tag
	kwp    feature.KeywordProvider
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = Tag
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) SetKeywordProvider(tag feature.Tag) MakeFeature {
	f.kwpTag = tag
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

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// reqArgv := site.GetRequestArgv(r)
	if p.Type() == "random" {

		if v, ok := p.Context().Get("Random").(string); !ok {
			log.ErrorF("random page without random key: %v", p.Url())
			redirect = "/random"
			return
		} else {
			switch v {
			case "a", "author":
				author := f.getRandomAuthor()
				authorKey := strcase.ToSnake(author)
				p.Context().SetSpecific("AuthorKey", authorKey)
				p.Context().SetSpecific("AuthorName", author)
				p.Context().SetSpecific("MetaRefresh", "5; url=/a/"+url.PathEscape(authorKey))
			case "t", "topic":
				topic := f.getRandomTopic()
				p.Context().SetSpecific("Topic", topic)
				p.Context().SetSpecific("MetaRefresh", "5; url=/t/"+url.PathEscape(topic))
			case "q", "quote":
				quoteUrl := f.getRandomQuoteUrl()
				if quotePg := f.Enjin.FindPage(f.Enjin.SiteDefaultLanguage(), quoteUrl); quotePg != nil {
					p.Context().SetSpecific("QuoteUrl", quotePg.Url())
					quoteHash, _ := quotePg.Context().Get("QuoteHash").(string)
					p.Context().SetSpecific("QuoteHash", quoteHash)
					p.Context().SetSpecific("MetaRefresh", "5; url="+quotePg.Url())
				} else {
					log.ErrorF("error finding page by random quote url: %v", quoteUrl)
					redirect = "/r/q/"
					return
				}
			case "w", "word":
				word := f.getRandomWord()
				p.Context().SetSpecific("Word", word)
				p.Context().SetSpecific("MetaRefresh", "5; url=/w/"+url.PathEscape(word))
			default:
				log.ErrorF("random page with invalid random key value: %v", v)
				redirect = "/random"
				return
			}
		}

		pg = p
		processed = true
	}
	return
}

func (f *CFeature) getRandomAuthor() (topic string) {
	results := f.Enjin.SelectQL(`SELECT DISTINCT .QuoteAuthor`)
	if v, ok := results["QuoteAuthor"]; ok {
		if vlist, ok := v.([]interface{}); ok {
			idx := rand.Intn(len(vlist))
			if topic = vlist[idx].(string); !ok {
				log.ErrorF("author not a string?! %#v", vlist[idx])
			}
		}
	}
	return
}

func (f *CFeature) getRandomTopic() (topic string) {
	results := f.Enjin.SelectQL(`SELECT DISTINCT .QuoteCategories`)
	if v, ok := results["QuoteCategories"]; ok {
		if vlist, ok := v.([]interface{}); ok {
			idx := rand.Intn(len(vlist))
			if topic = vlist[idx].(string); !ok {
				log.ErrorF("topic not a string?! %#v", vlist[idx])
			}
		}
	}
	return
}

func (f *CFeature) getRandomQuoteUrl() (quoteUrl string) {
	// TODO: optimize getRandomQuoteUrl
	getRandomUrl := func() (grUrl string) {
		selected := f.Enjin.SelectQL(`SELECT RANDOM .Url`)
		if v, ok := selected["Url"]; ok {
			if vs, ok := v.(string); ok {
				grUrl = vs
			}
		}
		return
	}
	quoteUrl = getRandomUrl()
	tries := 10
	for quoteUrl == "" || !strings.HasPrefix(quoteUrl, "/q/") {
		quoteUrl = getRandomUrl()
		if tries -= 1; tries <= 0 {
			break
		}
	}
	return
}

func (f *CFeature) getRandomWord() (word string) {
	idx := rand.Intn(f.kwp.Size())
	counter := 0
	f.kwp.Range(func(keyword string, _ []string) (proceed bool) {
		if counter < idx {
			counter += 1
			return true
		}
		word = keyword
		return false
	})
	return
}