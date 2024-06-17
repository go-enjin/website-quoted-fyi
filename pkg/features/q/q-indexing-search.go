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

package q

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/blevesearch/bleve/v2/mapping"

	clStrings "github.com/go-corelibs/strings"
	"github.com/go-corelibs/x-text/language"
	"github.com/go-enjin/be/pkg/feature"
)

func (f *CFeature) SearchDocumentMapping(tag language.Tag) (doctype string, dm *mapping.DocumentMapping) {
	doctype, _, dm = f.NewDocumentMapping(tag)
	return
}

func (f *CFeature) AddSearchDocumentMapping(tag language.Tag, indexMapping *mapping.IndexMappingImpl) {
	doctype, dm := f.SearchDocumentMapping(tag)
	indexMapping.AddDocumentMapping(doctype, dm)
}

func (f *CFeature) IndexDocument(pg feature.Page) (out interface{}, err error) {

	var rendered string

	if strings.HasSuffix(pg.Format(), ".tmpl") {
		var buf bytes.Buffer
		if tt, e := template.New("content.q.tmpl").
			Funcs(f.Enjin.MakeFuncMap(f.Enjin.Context(nil)).AsHTML()).
			Parse(pg.Content()); e != nil {
			err = fmt.Errorf("error parsing template: %v", e)
			return
		} else if e = tt.Execute(&buf, pg.Context()); e != nil {
			err = fmt.Errorf("error executing template: %v", e)
			return
		} else {
			rendered = buf.String()
		}
	} else {
		rendered = pg.Content()
	}

	doc := NewQuoteDocument(pg.Language(), pg.Url(), pg.Title())

	doc.SetAuthor(pg.Context().String("QuoteAuthor", ""))
	doc.AddCategory(pg.Context().Strings("QuoteCategories")...)

	if !clStrings.Empty(rendered) {
		doc.AddContent(rendered)
	}

	out = doc
	err = nil
	return
}
