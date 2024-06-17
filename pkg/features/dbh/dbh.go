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

	"github.com/urfave/cli/v2"

	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "qf-indexing-dbh"

type Feature interface {
	feature.Feature
	feature.PageTypeProcessor

	EQL() feature.QueryIndexFeature

	GetWidFrom(word string) (wid int64, ok bool)
	GetWordFrom(wid int64) (word string, ok bool)

	GetWordInfoFrom(words ...string) (wordList []string, lookupWid map[string]int64, lookupFlat, lookupWord map[string]string)

	GetIndexedBuilderKey(human string) (widString string)
	GetHumanBuilderKey(widString string) (human string)
	GetShortestBuilderKey(full string) (shortest string, ok bool)

	GetBuilderKeyCount(prefix string) (count int, ok bool)
	GetBuilderKeyQuotes(prefix string) (keys []string, quotes []*quote.Quote)
	GetBuilderKeyFor(shasum string) (key string, ok bool)

	GetNextBuilderKeyWords(prefix string) (words []string)

	GetFirstWords(prefix string) (words []string)
	GetSecondWords(prefix string, first int) (words []string)
	GetFirstWordLetters() (letters []string)

	GetWordShasums(word string) (shasums []string)
	GetWordPageStubs(word string) (stubs []*feature.PageStub)
	GetPaginatedWordPageStubs(word string, pg, size int) (stubs []*feature.PageStub, total int)
	GetPaginatedWordShasums(word string, pg, size int) (shasums []string)

	GetAuthorNames(nameOrKey string) (fullName, lastName, key string, ok bool)
	GetAuthorKeyFrom(name string) (key string, ok bool)
	GetAuthorNameFrom(name string) (key string, ok bool)

	GetTopicNames(nameOrKey string) (topic, key string, ok bool)
	GetTopicKeyFrom(nameOrKey string) (key string, ok bool)
	GetTopicNameFrom(nameOrKey string) (key string, ok bool)

	GetRandomWord() (word string)
	GetRandomTopic() (topic string)
	GetRandomAuthor() (name string)
	GetRandomQuote() (shasum string)
	GetRandomQuoteUrl() (url, hash string)

	TotalWords() int64
	TotalQuotes() int64
	TotalTopics() int64
	TotalAuthors() int64
	TotalFirstWords() int64
	TotalSecondWords() int64
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature

	eql feature.QueryIndexFeature

	firstWordLetters []string

	numWords       int64
	numQuotes      int64
	numTopics      int64
	numAuthors     int64
	numFirstWords  int64
	numSecondWords int64
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

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	if err = f.CFeature.Startup(ctx); err != nil {
		return
	}

	if found := f.Enjin.GetQueryIndexFeatures(); len(found) > 0 {
		f.eql = found[0]
	} else {
		err = fmt.Errorf("%v feature requires at least one feature.QueryIndexFeature", f.Tag())
		return
	}

	return
}

func (f *CFeature) EQL() feature.QueryIndexFeature {
	return f.eql
}
