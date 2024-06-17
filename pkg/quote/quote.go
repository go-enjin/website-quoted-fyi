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

package quote

import (
	"strings"

	"github.com/go-corelibs/rxp"
	"github.com/go-corelibs/shasum"
	clStrings "github.com/go-corelibs/strings"
)

type Quote struct {
	Url  string
	Hash string
}

type Author struct {
	Url  string
	Key  string
	Name string
	Last string

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
	key = GetFirstCharacters(1, clStrings.LastName(input))
	return
}

func GetLastNameKey(input string) (key string) {
	key = GetFirstCharacters(3, clStrings.LastName(input))
	return
}

var (
	// rxFlatten is an rxp version of the FlattenContent process.
	// The original process converted spaces after the non-word characters,
	// which was actually a useless step. This version converts spaces first
	// and then converts the actual non-word character range to underscores
	rxFlatten = rxp.Pipeline{
		{Transform: strings.TrimSpace},
		{
			Search:  rxp.Pattern{rxp.S("+")},
			Replace: rxp.Replace[string]{}.WithLiteral("_"),
		},
		{
			Search:  rxp.Pattern{rxp.R("-_a-zA-Z0-9", "^", "+")},
			Replace: rxp.Replace[string]{}.WithLiteral("_"),
		},
		{Transform: strings.ToLower},
	}
)

func FlattenContent(text string) string {
	//o := strings.TrimSpace(text)
	//o = rxNonWords.ReplaceAllString(o, "_")   // [^a-zA-Z0-9]
	//o = rxEmptySpace.ReplaceAllString(o, "_") // \s+
	//return strings.ToLower(o)
	return rxFlatten.Process(text)
}

func HashContent(content string) (hash string) {
	hash = shasum.Sha1Sum([]byte(FlattenContent(content)))
	return
}
