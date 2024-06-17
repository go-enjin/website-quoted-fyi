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
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/go-corelibs/context"
	"github.com/go-corelibs/rxp"
	"github.com/go-corelibs/slices"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var rxFieldWord = rxp.Pattern{rxp.IsFieldWord("c")}

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
	if author, ok = data["A"].(string); !ok {
		err = fmt.Errorf("missing quote author: %v - %#+v", ctx.Get("Url"), data)
		return
	}

	var categoryKeys, categories []string
	var v []interface{}
	if v, ok = data["C"].([]interface{}); !ok {
		err = fmt.Errorf("missing quote categories: %v", ctx.Get("Url"))
		return
	}
	for _, vv := range v {
		if vs, ok := vv.(string); ok {
			vsl := strings.ToLower(vs)
			if !slices.Within(vsl, categories) {
				categories = append(categories, vsl)
				categoryKeys = append(categoryKeys, quote.FlattenContent(vsl))
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
	ctx.SetSpecific("QuoteAuthorKey", quote.FlattenContent(author))
	//ctx.SetSpecific("QuoteSummary", description)
	ctx.SetSpecific("QuoteCategories", categories)
	ctx.SetSpecific("QuoteCategoryKeys", categoryKeys)

	out = ctx
	return
}

func (f *CFeature) Process(ctx context.Context, content string) (html template.HTML, redirect string, err error) {
	content = strings.TrimSpace(content)
	scheme := "https"
	var host, pgUrl string
	if host = ctx.String(".Request.Host", ""); host == "" {
		scheme = "http"
		_, listener, port := f.Enjin.ServiceInfo()
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
	quoteBody := "Quote " + quoteHash + ":\n\n\"" + content + "\"\n\n -- " + quoteAuthor

	twitterUrl := "https://twitter.com/share"
	twitterUrl += "?url=" + quoteUrl
	twitterUrl += "&text=" + quoteBody
	twitterUrl += "&via=quoted_fyi"
	twitterUrl += "&hashtags=" + strings.Join(quoteCategories, ",") + ",QuotedFYI"
	ctx.SetSpecific("QuoteShareTwitterUrl", twitterUrl)

	emailUrl := "mailto:"
	emailUrl += "?subject=" + description
	emailUrl += "&body=" + quoteBody + "\n\n" + quoteUrl
	ctx.SetSpecific("QuoteShareEmailUrl", emailUrl)

	matches := rxFieldWord.FindAllStringSubmatchIndex(content, -1)
	input := []rune(content)
	var last int
	var buf strings.Builder
	for _, group := range matches {
		if last < group[0][0] {
			buf.WriteString(string(input[last:group[0][0]]))
		}
		last = group[0][1]
		var value string
		if len(input) < group[0][1] {
			value = string(input[group[0][0]:])

		} else {
			value = string(input[group[0][0]:group[0][1]])
		}
		buf.WriteString(fmt.Sprintf(
			`<a href="/w/%s">%s</a>`,
			quote.FlattenContent(value),
			value,
		))
	}
	html = template.HTML(buf.String())
	return
}
