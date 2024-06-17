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
	"fmt"
	"math"
	"strings"

	"github.com/erni27/imcache"

	"github.com/go-corelibs/enjinql"
	"github.com/go-corelibs/rxp"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

type intHash struct{}

func (h intHash) Sum64(k int64) uint64 {
	return uint64(k)
}

type cache struct {
	lookupWID  *imcache.Sharded[string, int64]
	lookupWORD *imcache.Sharded[int64, string]
	uniqueFWID *imcache.Sharded[int64, bool]
	uniqueSWID *imcache.Sharded[string, bool]
}

var (
	gStartupCache = &cache{
		lookupWID: imcache.NewSharded[string, int64](
			1000,
			imcache.DefaultStringHasher64{},
		),
		lookupWORD: imcache.NewSharded[int64, string](
			1000,
			intHash{},
		),
		uniqueFWID: imcache.NewSharded[int64, bool](
			1000,
			intHash{},
		),
		uniqueSWID: imcache.NewSharded[string, bool](
			1000,
			imcache.DefaultStringHasher64{},
		),
	}
)

func (f *CFeature) AddSources() (sources enjinql.ConfigSources) {
	return enjinql.ConfigSources{

		// topic name(s)
		enjinql.MakeSourceConfig(
			"",
			quote.WordSource,
			enjinql.NewStringValue("flat", 256),
			enjinql.NewStringValue("letter", 1),
			enjinql.NewStringValue("word", 256),
		).
			AddUnique("flat").
			AddIndex("flat").
			AddIndex("letter").
			AddIndex("word").
			AddIndex("flat", "word").
			AddIndex("word", "flat").
			AddIndex("letter", "word", "flat").
			AddIndex("flat", "word", "letter"),

		// joining pages with word
		enjinql.MakeSourceConfig(
			enjinql.PageSource,
			quote.PageWordSource,
			enjinql.NewLinkedValue(quote.WordSource, "id"),
			enjinql.NewIntValue("tally"),
		).
			AddIndex("page_id").
			AddIndex("word_id").
			AddIndex("tally").
			AddIndex("page_id", "word_id", "tally").
			AddIndex("word_id", "page_id", "tally"),

		// first words
		enjinql.MakeSourceConfig(
			quote.WordSource,
			quote.FirstWordSource,
			// primary nil is okay because there is a parent
			// and the purpose is simply a word_id list
			nil,
		).
			AddIndex("word_id"),

		// second words
		enjinql.MakeSourceConfig(
			quote.WordSource,
			quote.SecondWordSource,
			enjinql.NewLinkedValue(quote.FirstWordSource, "id"),
		).
			AddIndex("word_id").
			AddIndex("first_word_id").
			AddIndex("word_id", "first_word_id").
			AddIndex("first_word_id", "word_id"),

		// builder keys
		enjinql.MakeSourceConfig(
			enjinql.PageSource,
			quote.BuilderKeySource,
			enjinql.NewStringValue("key", -1),
		).
			AddIndex("page_id").
			AddIndex("key").
			AddIndex("page_id", "key").
			AddIndex("key", "page_id"),
	}
}

var (
	rxFieldWord = rxp.Pattern{rxp.IsFieldWord("c")}
)

func (f *CFeature) AddToSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	if p.Type() != "quote" {
		return // only quotes have words
	}
	var flats, words []string

	tally := make(map[string]int)
	content := strings.TrimSpace(p.Content())
	fields := rxFieldWord.FindAllString(strings.ToLower(content), -1)

	for _, field := range fields {
		flat := quote.FlattenContent(field)
		if _, present := tally[flat]; !present {
			words = append(words, field)
			flats = append(flats, flat)
		}
		tally[flat] += 1
	}

	var fWid int64
	for idx, word := range words {
		var flat, letter string
		letter = strings.ToLower(string(word[0]))
		flat = flats[idx] // same idx as words

		var wid int64 = math.MaxInt64
		if found, present := gStartupCache.lookupWID.Get(flat); present && found != math.MaxInt64 {
			wid = found
		} else {

			if wid, err = tx.Insert(quote.WordSource, flat, letter, word); err != nil {
				return
			}

			gStartupCache.lookupWID.Set(flat, wid, imcache.WithNoExpiration())
			gStartupCache.lookupWORD.Set(wid, flat, imcache.WithNoExpiration())
		}

		if _, err = tx.Insert(quote.PageWordSource, sid, wid, tally[flat]); err != nil {
			return
		}

		switch idx {
		case 0: // first words data source
			fWid = wid
			if present, _ := gStartupCache.uniqueFWID.Get(wid); !present {
				if _, err = tx.Insert(quote.FirstWordSource, wid); err != nil {
					return
				}
				gStartupCache.uniqueFWID.Set(wid, true, imcache.WithNoExpiration())
			}

		case 1: // first+second words data source
			pair := fmt.Sprintf("%d-%d", fWid, wid)
			if present, _ := gStartupCache.uniqueSWID.Get(pair); !present {
				if _, err = tx.Insert(quote.SecondWordSource, wid, fWid); err != nil {
					return
				}
				gStartupCache.uniqueSWID.Set(pair, true, imcache.WithNoExpiration())
			}
		}
	}

	var bids []string
	for _, flat := range flats {
		if wid, ok := gStartupCache.lookupWID.Get(flat); ok {
			bids = append(bids, fmt.Sprintf("%d", wid))
		}
	}
	_, err = tx.Insert(quote.BuilderKeySource, sid, strings.Join(bids, "-"))
	return
}

func (f *CFeature) RemoveFromSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	// nop, read-only site!
	return
}
