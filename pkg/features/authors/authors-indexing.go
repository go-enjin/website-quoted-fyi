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
	"math"
	"strings"

	"github.com/erni27/imcache"

	"github.com/go-corelibs/enjinql"
	clStrings "github.com/go-corelibs/strings"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	gStartupCache = imcache.NewSharded[string, int64](
		1000,
		imcache.DefaultStringHasher64{},
	)
)

func (f *CFeature) AddSources() (sources enjinql.ConfigSources) {
	return enjinql.ConfigSources{
		// author name(s)
		enjinql.MakeSourceConfig(
			"",
			quote.AuthorSource,
			enjinql.NewStringValue("flat", 512),
			enjinql.NewStringValue("letter", 1),
			enjinql.NewStringValue("last_name", 512),
			enjinql.NewStringValue("full_name", 512),
		).
			AddUnique("flat").
			AddIndex("flat").
			AddIndex("letter").
			AddIndex("last_name").
			AddIndex("full_name").
			AddIndex("full_name", "flat").
			AddIndex("flat", "full_name").
			AddIndex("letter", "flat", "full_name").
			AddIndex("flat", "full_name", "letter"),
		// joining pages with author
		enjinql.MakeSourceConfig(
			enjinql.PageSource,
			quote.PageAuthorSource,
			enjinql.NewLinkedValue(quote.AuthorSource, "id"),
		).
			AddIndex("page_id").
			AddIndex("author_id").
			AddIndex("author_id", "page_id").
			AddIndex("page_id", "author_id"),
	}
}

func (f *CFeature) AddToSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	if p.Type() != "quote" {
		return // only quotes have authors
	}
	var flat, letter, lastName, fullName string

	if fullName = p.Context().String("QuoteAuthor", ""); fullName == "" {
		log.ErrorF("quote is missing .QuoteAuthor: %q", stub.Source)
		return // just skip, not an actual error
	} else if lastName = clStrings.LastName(fullName); lastName == "" {
		log.ErrorF(".QuoteAuthor has no last name: %q - %q", fullName, stub.Source)
		return // just skip, not an actual error
	}
	letter = strings.ToLower(string(lastName[0]))
	flat = quote.FlattenContent(fullName)

	var aid int64 = math.MaxInt64

	if found, present := gStartupCache.Get(flat); present {
		aid = found
	} else {

		// not present in startup cache means not present in the database, no
		// need to be overly cautious and check if the entries exist first

		if aid, err = tx.Insert(quote.AuthorSource, flat, letter, lastName, fullName); err != nil {
			return
		}

		gStartupCache.Set(flat, aid, imcache.WithNoExpiration())
	}

	_, err = tx.Insert(quote.PageAuthorSource, sid, aid)
	return
}

func (f *CFeature) RemoveFromSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	// nop, read-only site!
	return
}
