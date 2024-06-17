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
	"math"
	"sort"
	"strings"

	"github.com/maruel/natural"

	"github.com/go-enjin/be/pkg/feature"
)

func (f *CFeature) GetWordInfoFrom(inputs ...string) (wordList []string, lookupWid map[string]int64, lookupFlat, lookupWord map[string]string) {
	lookupFlat = make(map[string]string)
	lookupWord = make(map[string]string)
	lookupWid = make(map[string]int64)
	if size := len(inputs); size == 0 {
		return
	} else {
		wordList = make([]string, size)
	}

	var argv []interface{}
	var qq string
	for idx, word := range inputs {
		word = strings.ToLower(word)
		inputs[idx] = word
		wordList[idx] = word
		if idx > 0 {
			qq += ", "
		}
		qq += "%q"
		argv = append(argv, word)
	}
	query := `LOOKUP word.ID, word.Flat, word.Word WITHIN (word.Flat in (` + qq + `)) OR (word.Word IN (` + qq + `));`

	var args []interface{}
	args = append(args, argv...)
	args = append(args, argv...)
	if _, results, err := f.eql.PerformLookup(query, args...); err == nil {
		for _, result := range results {
			if flat := result.String("flat"); flat != "" {
				word := result.String("word")
				lookupFlat[word] = flat
				lookupWord[flat] = word
				lookupWid[flat] = result.Int64("id")
			}
		}
	}

	for idx, input := range inputs {
		if word, present := lookupWord[input]; present {
			wordList[idx] = word
		} else if flat, ok := lookupFlat[input]; ok {
			wordList[idx] = lookupWord[flat]
		} else {
			// this can happen because the caller may have supplied words
			// that don't actually exist in the database!
			wordList[idx] = ""
		}
	}

	return
}

func (f *CFeature) GetWords2Wids(words ...string) (words2wids map[string]int64) {
	if len(words) == 0 {
		return
	}

	words2wids = make(map[string]int64)

	var argv []interface{}
	var qq string
	for idx, word := range words {
		if idx > 0 {
			qq += ", "
		}
		qq += "%q"
		argv = append(argv, word)
	}
	query := `LOOKUP word.ID, word.Word WITHIN (word.Flat in (` + qq + `)) OR (word.Word IN (` + qq + `));`

	if _, results, err := f.eql.PerformLookup(query, append(argv, argv...)); err == nil {
		for _, result := range results {
			if id := result.ValueAsInt64("id"); id != math.MaxInt64 {
				word := result.String("word")
				words2wids[word] = id
			}
		}
	}

	return
}

func (f *CFeature) GetWids2Words(wids ...int64) (wids2words map[int64]string) {
	if len(wids) == 0 {
		return
	}

	wids2words = make(map[int64]string)

	var argv []interface{}
	var qq string
	for idx, word := range wids {
		if idx > 0 {
			qq += ", "
		}
		qq += "%q"
		argv = append(argv, word)
	}
	query := `LOOKUP word.ID, word.Word WITHIN word.ID IN (` + qq + `);`

	if _, results, err := f.eql.PerformLookup(query, argv...); err == nil {
		for _, result := range results {
			if id := result.ValueAsInt64("id"); id != math.MaxInt64 {
				wids2words[id] = result.String("word")
			}
		}
	}

	return
}

func (f *CFeature) GetWords(wids ...int64) (words []string) {
	if len(wids) == 0 {
		return
	}

	var argv []interface{}
	var qq string
	for idx, word := range wids {
		if idx > 0 {
			qq += ", "
		}
		qq += "%d"
		argv = append(argv, word)
	}
	query := `LOOKUP word.Word WITHIN word.ID IN (` + qq + `);`

	if _, results, err := f.eql.PerformLookup(query, argv...); err == nil {
		for _, result := range results {
			words = append(words, result.String("word"))
		}
	}

	return
}

func (f *CFeature) GetWidFrom(word string) (wid int64, ok bool) {
	//wid, ok = f.lookupWidFromWord.Load(word)

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP word.ID WITHIN (word.Flat == %q) OR (word.Word == %q) LIMIT 1`,
		word, word,
	); err != nil {
		return
	} else {
		v := results.FirstInt64Value("id")
		if ok = v != math.MinInt64; ok {
			wid = v
		}
	}

	return
}

func (f *CFeature) GetWordFrom(wid int64) (word string, ok bool) {
	if _, results, err := f.eql.PerformLookup(`LOOKUP word.Word WITHIN word.ID == %d`, wid); err != nil {
		return
	} else {
		word = results.FirstStringValue("word")
		ok = word != ""
	}
	return
}

func (f *CFeature) GetWordPageStubs(word string) (stubs []*feature.PageStub) {
	stubs, _ = f.eql.PerformQuery(`QUERY WITHIN (word.Flat == %q) OR (word.Word == %q) ORDER BY .ID`, word, word)
	return
}

func (f *CFeature) GetPaginatedWordPageStubs(word string, pg, size int) (stubs []*feature.PageStub, total int) {

	if _, results, err := f.eql.PerformLookup(`LOOKUP COUNT .Shasum AS total WITHIN (word.Flat == %q) OR (word.Word == %q)`, word, word); err == nil {
		if v := results.FirstInt64Value("total"); v != math.MinInt64 {
			total = int(v)
			stubs, _ = f.eql.PerformQuery(
				`QUERY WITHIN (word.Flat == %q) OR (word.Word == %q) ORDER BY .ID OFFSET %d LIMIT %d`,
				word, word,
				pg*size, size,
			)
		}
	}

	return
}

func (f *CFeature) GetWordShasums(word string) (shasums []string) {

	if _, results, err := f.eql.PerformLookup(`LOOKUP DISTINCT .Shasum WITHIN (word.Flat == %q) OR (word.Word == %q) ORDER BY .ID`, word, word); err == nil {
		shasums = results.StringValues("shasum")
	}

	return
}

func (f *CFeature) GetPaginatedWordShasums(word string, pg, size int) (shasums []string) {

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP DISTINCT .Shasum WITHIN (word.Flat == %q) OR (word.Word == %q) ORDER BY .ID OFFSET %d LIMIT %d`,
		word, word,
		pg*size, size,
	); err == nil {
		shasums = results.StringValues("shasum")
	}

	return
}

func (f *CFeature) GetFirstWords(letter string) (words []string) {

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP DISTINCT word.Word WITHIN (word.Letter == %q) AND (first_word.WordID >= 0) ORDER BY word.ID`,
		letter,
	); err == nil {
		words = results.StringValues("word")
	}

	return
}

func (f *CFeature) GetSecondWords(letter string, first int) (words []string) {

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP DISTINCT word.Word WITHIN second_word.FirstWordID == %d ORDER BY word.ID`,
		first,
	); err == nil {
		words = results.StringValues("word")
	}

	return
}

func (f *CFeature) GetFirstWordLetters() (letters []string) {
	if len(f.firstWordLetters) > 0 {
		letters = f.firstWordLetters[:]
		return
	}

	if _, results, err := f.eql.PerformLookup(
		`LOOKUP DISTINCT word.Letter WITHIN first_word.WordID >= 0`,
	); err == nil {
		letters = results.StringValues("letter")
		sort.Sort(natural.StringSlice(letters))
		f.firstWordLetters = letters
	}

	return
}
