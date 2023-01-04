// Copyright (c) 2022  The Go-Enjin Authors
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

package quote

import (
	"strings"

	"github.com/go-enjin/be/pkg/hash/sha"
	"github.com/go-enjin/be/pkg/regexps"
	beStrings "github.com/go-enjin/be/pkg/strings"
)

type Quote struct {
	Url  string
	Hash string
}

type Author struct {
	Url  string
	Key  string
	Name string

	Quotes []*Quote
}

type AuthorsGroup struct {
	Key     string
	Authors []*Author
}

type QuotesGroup struct {
	Key    string
	Quotes []*Quote
}

type QuotesGroups struct {
	Key    string
	Groups []*QuotesGroup
}

type TopicAuthors struct {
	Key     string
	Name    string
	Authors []*Author
}

type TopicAuthorsGroup struct {
	Key    string
	Topics []*TopicAuthors
}

type TopicQuotes struct {
	Key    string
	Name   string
	Quotes []*Quote
}

type WordTopicGroup struct {
	Key    string
	Topics []*TopicQuotes
}

type WordGroup struct {
	Key   string
	Words []string
}

type WordLink struct {
	Path string
	Word string
}

func GetFirstCharacters(num int, word string) (key string) {
	if len(word) >= num {
		key = strings.ToLower(word[:num])
	} else {
		key = strings.ToLower(word)
	}
	return
}

func GetLastNameCharacter(input string) (key string) {
	key = GetFirstCharacters(1, beStrings.LastName(input))
	return
}

func GetLastNameKey(input string) (key string) {
	key = GetFirstCharacters(3, beStrings.LastName(input))
	return
}

func FlattenContent(text string) string {
	o := strings.TrimSpace(text)
	o = regexps.RxNonWord.ReplaceAllString(o, "_")
	o = regexps.RxEmptySpace.ReplaceAllString(o, "_")
	return strings.ToLower(o)
}

func HashContent(content string) (hash string) {
	hash = sha.DataHashSha1([]byte(FlattenContent(content)))
	return
}