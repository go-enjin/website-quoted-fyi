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

package dbh

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/maruel/natural"

	"github.com/go-corelibs/maps"
	"github.com/go-corelibs/maths"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) GetBuilderKeyCount(prefix string) (count int, ok bool) {

	_, results, err := f.eql.PerformLookup(
		`LOOKUP COUNT builder.ID AS total WITHIN (builder.Key == %q) OR (builder.Key LIKE %q)`,
		prefix, prefix+"-%",
	)
	if ok = err == nil; ok {
		count = results.FirstIntValue("total")
	}

	return
}

func (f *CFeature) GetBuilderKeyQuotes(prefix string) (keys []string, quotes []*quote.Quote) {

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP .Url, quote.Hash, builder.Key WITHIN (builder.Key == %q) OR (builder.Key LIKE %q)`,
		prefix, prefix+"-%",
	); err != nil {
		log.ErrorF("error looking up builder key quotes: %q - %v", prefix, err)
	} else {
		for _, result := range results {
			if url := result.String("url"); url != "" {
				if hash := result.String("hash"); hash != "" {
					if bkey := result.String("key"); bkey != "" {
						keys = append(keys, bkey)
						quotes = append(quotes, &quote.Quote{
							Url:  url,
							Hash: hash,
						})
					}
				}
			}
		}
	}

	return
}

func (f *CFeature) GetShortestBuilderKey(full string) (shortest string, ok bool) {
	var wids []int
	for _, value := range strings.Split(full, "-") {
		wid, _ := strconv.Atoi(value)
		wids = append(wids, wid)
	}
	half := maths.Ceil(len(wids)/2, 5)
	if count, found := f.GetBuilderKeyCount(Join(wids[:half], "-")); found {
		if count <= 2 {
			shortest, ok = f.findShortestKeyRev(half, wids)
		} else {
			shortest, ok = f.findShortestKeyFwd(half, wids)
		}
	}
	return
}

func (f *CFeature) findShortestKeyRev(start int, full []int) (shortest string, ok bool) {
	this := start + 1
	for {
		if this -= 1; this <= 0 {
			shortest = fmt.Sprintf("%v", full[0])
			break
		}
		shortest = Join(full[:this], "-")
		count, found := f.GetBuilderKeyCount(shortest)
		if ok = found && count > 2; ok {
			break
		}
	}
	return
}

func (f *CFeature) findShortestKeyFwd(start int, full []int) (shortest string, ok bool) {
	last := len(full) - 1
	this := start - 1
	for {
		if this += 1; this >= last {
			break
		}
		shortest = Join(full[:this], "-")
		count, found := f.GetBuilderKeyCount(shortest)
		if ok = found && count <= 2; ok {
			// searching forwards found singularity
			// the shortest path is the previous iteration
			shortest = Join(full[:this-1], "-")
			break
		}
	}
	return
}

func (f *CFeature) GetHumanBuilderKey(widString string) (human string) {
	var words []string
	for _, value := range strings.Split(widString, "-") {
		if wid, err := strconv.ParseInt(value, 10, 64); err != nil {
			log.ErrorF("error converting string to wid value: %q, err", value, err)
		} else {
			if word, ok := f.GetWordFrom(wid); ok {
				words = append(words, quote.FlattenContent(word))
			} else {
				words = append(words, "ERROR")
			}
		}
	}
	human = strings.Join(words, "-")
	return
}

func (f *CFeature) GetIndexedBuilderKey(human string) (indexed string) {
	var wids []string
	for _, word := range strings.Split(human, "-") {
		if wid, ok := f.GetWidFrom(word); ok {
			wids = append(wids, fmt.Sprintf("%d", wid))
		} else {
			wids = append(wids, "err")
		}
	}
	indexed = Join(wids, "-")
	return
}

func (f *CFeature) GetBuilderKeyFor(shasum string) (key string, ok bool) {

	_, results, err := f.eql.PerformLookup(`LOOKUP builder.Key WITHIN .Shasum == %q LIMIT 1`, shasum)
	if ok = err == nil; ok {
		key = results.FirstStringValue("key")
	}

	return
}

func (f *CFeature) GetNextBuilderKeyWords(prefix string) (words []string) {

	unique := make(map[int64]struct{})

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP DISTINCT builder.Key WITHIN builder.Key LIKE %q`,
		prefix+"-%",
	); err != nil {
		return
	} else {
		for _, result := range results {
			if trimmed := strings.TrimPrefix(result.String("key"), prefix+"-"); trimmed != "" {
				if widstr, _, _ := strings.Cut(trimmed, "-"); widstr != "" {
					if i, err := strconv.ParseInt(widstr, 10, 64); err == nil {
						unique[i] = struct{}{}
					}
				}
			}
		}
	}

	wids := maps.Keys(unique)
	words = f.GetWords(wids...)
	sort.Sort(natural.StringSlice(words))

	return
}
