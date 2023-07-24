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

package q

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/go-enjin/golang-org-x-text/language"
	"github.com/iancoleman/strcase"

	"github.com/go-enjin/be/pkg/context"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/indexing/search"
	"github.com/go-enjin/be/pkg/page"
	"github.com/go-enjin/be/pkg/regexps"
	beStrings "github.com/go-enjin/be/pkg/strings"
	"github.com/go-enjin/be/pkg/theme"
	types "github.com/go-enjin/be/pkg/types/theme-types"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

const (
	Tag feature.Tag = "quote-page-format"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

func init() {
	search.RegisterSearchPageType("quote")
}

type Feature interface {
	feature.Feature
	types.Format
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

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Name() (name string) {
	name = "q"
	return
}

func (f *CFeature) Extensions() (extensions []string) {
	extensions = append(extensions, "q", "q.tmpl")
	return
}

func (f *CFeature) Label() (label string) {
	label = "Q"
	return
}

func (f *CFeature) Prepare(ctx context.Context, content string) (out context.Context, err error) {
	var ok bool
	var data map[string]interface{}
	if data, ok = ctx.Get("Q").(map[string]interface{}); !ok {
		err = fmt.Errorf("missing quote context: %v", ctx.Get("Url"))
		return
	}
	var author string
	if author, ok = data["a"].(string); !ok {
		err = fmt.Errorf("missing quote author: %v", ctx.Get("Url"))
		return
	}

	var categoryKeys, categories []string
	var v []interface{}
	if v, ok = data["c"].([]interface{}); !ok {
		err = fmt.Errorf("missing quote categories: %v", ctx.Get("Url"))
		return
	}
	for _, vv := range v {
		if vs, ok := vv.(string); ok {
			vsl := strings.ToLower(vs)
			if !beStrings.StringInSlices(vsl, categories) {
				categories = append(categories, vsl)
				categoryKeys = append(categoryKeys, strcase.ToSnake(vsl))
			}
		}
	}

	hash := quote.HashContent(content)

	description := "Quote " + hash[:8]
	if len(categories) > 0 {
		description += " on"
		last := len(categories) - 1
		for idx, category := range categories {
			if idx == 0 {
				description += " "
			} else if idx < last {
				description += ", "
			} else if idx == last {
				description += " and "
			}
			description += category
		}
	}
	description += " by " + author

	ctx.SetSpecific("Url", "/"+hash)
	ctx.SetSpecific("Title", description)
	ctx.SetSpecific("Description", description)

	ctx.SetSpecific("QuoteHash", hash[:8])
	ctx.SetSpecific("QuoteShasum", hash)
	ctx.SetSpecific("QuoteAuthor", author)
	ctx.SetSpecific("QuoteAuthorKey", strcase.ToSnake(author))
	ctx.SetSpecific("QuoteSummary", description)
	ctx.SetSpecific("QuoteCategories", categories)
	ctx.SetSpecific("QuoteCategoryKeys", categoryKeys)

	out = ctx
	return
}

func (f *CFeature) Process(ctx context.Context, t types.Theme, content string) (html htmlTemplate.HTML, redirect string, err *types.EnjinError) {
	scheme := "https"
	var host, pgUrl string
	if host = ctx.String(".Request.Host", ""); host == "" {
		scheme = "http"
		listener, port := f.Enjin.ServiceInfo()
		if listener == "" || listener == "0.0.0.0" {
			listener = "localhost"
		}
		host = listener + ":" + strconv.Itoa(port)
	}
	pgUrl = ctx.Get("Url").(string)
	description := ctx.Get("Description").(string)
	quoteHash := ctx.Get("QuoteHash").(string)
	quoteAuthor := ctx.Get("QuoteAuthor").(string)
	quoteCategories := ctx.Get("QuoteCategories").([]string)
	quoteUrl := scheme + "://" + host + pgUrl
	quoteBody := "Quote " + quoteHash + ":\n\n\"" + strings.TrimSpace(content) + "\"\n\n -- " + quoteAuthor

	twitterUrl := "https://twitter.com/share"
	twitterUrl += "?url=" + quoteUrl
	twitterUrl += "&text=" + quoteBody
	twitterUrl += "&via=quoted_fyi"
	twitterUrl += "&hastags=" + strings.Join(quoteCategories, ",") + ",QuotedFYI"
	ctx.SetSpecific("QuoteShareTwitterUrl", twitterUrl)

	emailUrl := "mailto:"
	emailUrl += "?subject=" + description
	emailUrl += "&body=" + quoteBody + "\n\n" + quoteUrl
	ctx.SetSpecific("QuoteShareEmailUrl", emailUrl)

	var words, tokens []string
	var modified string
	var currentWord string
	contentLength := len(content)
	for idx, char := range content {
		character := string(char)
		var nextChar string
		if idx < contentLength-1 {
			nextChar = string(content[idx+1])
		}
		if regexps.RxKeyword.MatchString(currentWord+character) || regexps.RxKeyword.MatchString(currentWord+character+nextChar) {
			currentWord += character
		} else {
			if currentWord != "" {
				token := fmt.Sprintf("{TOKEN%d}", len(words))
				words = append(words, currentWord)
				tokens = append(tokens, token)
				modified += token + character
				currentWord = ""
			} else {
				modified += character
			}
		}
	}
	for idx, word := range words {
		token := tokens[idx]
		lower := strings.ToLower(word)
		link := fmt.Sprintf(`<a href="/w/%v">%v</a>`, lower, word)
		modified = strings.Replace(modified, token, link, 1)
	}
	html = htmlTemplate.HTML(modified)
	return
}

func (f *CFeature) SearchDocumentMapping(tag language.Tag) (doctype string, dm *mapping.DocumentMapping) {
	doctype, _, dm = f.NewDocumentMapping(tag)
	return
}

func (f *CFeature) AddSearchDocumentMapping(tag language.Tag, indexMapping *mapping.IndexMappingImpl) {
	doctype, dm := f.SearchDocumentMapping(tag)
	indexMapping.AddDocumentMapping(doctype, dm)
}

func (f *CFeature) IndexDocument(thing interface{}) (out interface{}, err error) {
	pg, _ := thing.(*page.Page) // FIXME: this "thing" avoids package import loops

	var rendered string

	if strings.HasSuffix(pg.Format, ".tmpl") {
		var buf bytes.Buffer
		if tt, e := htmlTemplate.New("content.q.tmpl").Funcs(theme.DefaultFuncMap()).Parse(pg.Content); e != nil {
			err = fmt.Errorf("error parsing template: %v", e)
			return
		} else if e = tt.Execute(&buf, pg.Context); e != nil {
			err = fmt.Errorf("error executing template: %v", e)
			return
		} else {
			rendered = buf.String()
		}
	} else {
		rendered = pg.Content
	}

	doc := NewQuoteDocument(pg.Language, pg.Url, pg.Title)

	doc.SetAuthor(pg.Context.Get("QuoteAuthor").(string))
	doc.AddCategory(pg.Context.Get("QuoteCategories").([]string)...)

	if !beStrings.Empty(rendered) {
		doc.AddContent(rendered)
	}

	out = doc
	err = nil
	return
}