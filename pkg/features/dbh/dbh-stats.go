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
)

func (f *CFeature) TotalWords() int64 {
	if f.numWords > 0 {
		return f.numWords
	}

	// total number of words indexed
	if _, results, ee := f.eql.PerformLookup(`LOOKUP COUNT word.ID AS total WITHIN word.ID >= 0`); ee != nil || results.Len() == 0 {
		log.ErrorF("error getting count of words: %w", ee)
	} else {
		return results.FirstInt64Value("total")
	}

	return 0
}

func (f *CFeature) TotalQuotes() int64 {
	if f.numQuotes > 0 {
		return f.numQuotes
	}

	// total number of quotes indexed
	if _, results, ee := f.eql.PerformLookup(`LOOKUP COUNT .ID AS total WITHIN .Type == "quote"`); ee != nil || results.Len() == 0 {
		log.ErrorF("error getting count of quotes: %w", ee)
	} else {
		return results.FirstInt64Value("total")
	}

	return 0
}

func (f *CFeature) TotalTopics() int64 {
	if f.numTopics > 0 {
		return f.numTopics
	}

	// total number of topics indexed
	if _, results, ee := f.eql.PerformLookup(`LOOKUP COUNT topic.ID AS total WITHIN topic.ID >= 0`); ee != nil || results.Len() == 0 {
		log.ErrorF("error getting count of topics: %w", ee)
	} else {
		return results.FirstInt64Value("total")
	}

	return 0
}

func (f *CFeature) TotalAuthors() int64 {
	if f.numAuthors > 0 {
		return f.numAuthors
	}

	// total number of authors indexed
	if _, results, ee := f.eql.PerformLookup(`LOOKUP COUNT author.ID AS total WITHIN author.ID >= 0`); ee != nil || results.Len() == 0 {
		log.ErrorF("error getting count of authors: %w", ee)
	} else {
		return results.FirstInt64Value("total")
	}

	return 0
}

func (f *CFeature) TotalFirstWords() int64 {
	if f.numFirstWords > 0 {
		return f.numFirstWords
	}
	// total number of first words indexed
	if _, results, ee := f.eql.PerformLookup(`LOOKUP COUNT first_word.ID AS total WITHIN first_word.ID >= 0`); ee != nil || results.Len() == 0 {
		log.ErrorF("error getting count of first words: %w", ee)
	} else {
		return results.FirstInt64Value("total")
	}

	return 0
}

func (f *CFeature) TotalSecondWords() int64 {
	if f.numSecondWords > 0 {
		return f.numSecondWords
	}
	// total number of second words indexed
	if _, results, ee := f.eql.PerformLookup(`LOOKUP COUNT DISTINCT second_word.word_id AS total WITHIN second_word.ID >= 0`); ee != nil || results.Len() == 0 {
		log.ErrorF("error getting count of second words: %w", ee)
	} else {
		return results.FirstInt64Value("total")
	}

	return 0
}
