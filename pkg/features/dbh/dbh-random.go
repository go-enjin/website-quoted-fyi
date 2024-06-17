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
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/types/page"
)

func (f *CFeature) GetRandomWord() (word string) {

	if _, results, err := f.eql.PerformLookup(`LOOKUP word.Word ORDER BY random() LIMIT 1`); err != nil {
		log.ErrorF("error selecting random word: %v", err)
	} else if len(results) == 0 {
		log.ErrorF("error selecting random word: zero results")
	} else {
		word = results[0].String("Word")
	}

	return
}

func (f *CFeature) GetRandomQuote() (shasum string) {

	if _, results, err := f.eql.PerformLookup(`LOOKUP .Shasum WITHIN .Type == "quote" ORDER BY random() LIMIT 1`); err != nil {
		log.ErrorF("error selecting random shasum: %v", err)
	} else if len(results) == 0 {
		log.ErrorF("error selecting random shasum: zero results")
	} else {
		shasum = results[0].String("Shasum")
	}

	return
}

func (f *CFeature) GetRandomQuoteUrl() (url, hash string) {
	maxTries := 100
	for i := 0; i < maxTries; i++ {
		shasum := f.GetRandomQuote()
		if stub := f.Enjin.FindPageStub(shasum); stub == nil {
			//log.ErrorF("error finding page stub for: %q", shasum)
		} else if pg, err := page.NewPageFromStub(stub, f.Enjin.MustGetTheme(), f.Enjin.Context(nil)); err != nil {
			//log.ErrorF("error making page from stub %q: %v", shasum, err)
		} else {
			url = pg.Url()
			hash = pg.Context().String("QuoteHash")
			return
		}
	}
	log.ErrorF("random quote url not found!")
	return
}

func (f *CFeature) GetRandomAuthor() (name string) {

	if _, results, err := f.eql.PerformLookup(`LOOKUP author.FullName ORDER BY random() LIMIT 1`); err != nil {
		log.ErrorF("error selecting random author: %v", err)
	} else if len(results) == 0 {
		log.ErrorF("error selecting random author: zero results")
	} else {
		name = results[0].String("FullName")
	}

	return
}

func (f *CFeature) GetRandomTopic() (topic string) {

	if _, results, err := f.eql.PerformLookup(`LOOKUP topic.Topic ORDER BY random() LIMIT 1`); err != nil {
		log.ErrorF("error selecting random topic: %v", err)
	} else if len(results) == 0 {
		log.ErrorF("error selecting random topic: zero results")
	} else {
		topic = results[0].String("Topic")
	}

	return
}
