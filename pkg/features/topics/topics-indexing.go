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
	"math"
	"strings"

	"github.com/erni27/imcache"

	"github.com/go-corelibs/enjinql"
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
		// topic name(s)
		enjinql.MakeSourceConfig(
			"",
			quote.TopicSource,
			enjinql.NewStringValue("flat", 512),
			enjinql.NewStringValue("letter", 1),
			enjinql.NewStringValue("topic", 512),
		).
			AddUnique("flat").
			AddIndex("flat").
			AddIndex("letter").
			AddIndex("topic").
			AddIndex("topic", "flat").
			AddIndex("flat", "topic").
			AddIndex("topic", "flat", "letter").
			AddIndex("letter", "topic", "flat"),
		// joining pages with topic
		enjinql.MakeSourceConfig(
			enjinql.PageSource,
			quote.PageTopicSource,
			enjinql.NewLinkedValue(quote.TopicSource, "id"),
		).
			AddIndex("page_id").
			AddIndex("topic_id").
			AddIndex("topic_id", "page_id").
			AddIndex("page_id", "topic_id"),
	}
}

func (f *CFeature) AddToSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	if p.Type() != "quote" {
		return // only quotes have topics
	}
	var flat, letter string
	var topics []string

	if topics = p.Context().Strings("QuoteCategories"); len(topics) == 0 {
		log.ErrorF("quote is missing .QuoteCategories: %q", stub.Source)
		return // just skip, not an actual error
	}

	for _, topic := range topics {
		if topic == "" {
			continue
		}
		letter = strings.ToLower(string(topic[0]))
		flat = quote.FlattenContent(topic)

		var tid int64 = math.MinInt64

		if found, present := gStartupCache.Get(flat); present {
			tid = found
		} else {

			if tid, err = tx.Insert(quote.TopicSource, flat, letter, topic); err != nil {
				return
			}
			gStartupCache.Set(flat, tid, imcache.WithNoExpiration())
		}

		if _, err = tx.Insert(quote.PageTopicSource, sid, tid); err != nil {
			return
		}
	}

	return
}

func (f *CFeature) RemoveFromSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	// nop, read-only site!
	return
}
