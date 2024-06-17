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
	"net/http"
	"sort"

	"github.com/maruel/natural"

	"github.com/go-corelibs/maps"
	clStrings "github.com/go-corelibs/strings"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) PageTypeNames() (names []string) {
	names = append(names, "authors", "author")
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {

	switch p.Type() {
	case "authors":
		pg, redirect, processed, err = f.ProcessGroupsPageType(r, p)
	case "author":
		//pg, redirect, processed, err = f.ProcessSinglePageType(r, p)
	default:
		// moved to dbh
		p.Context().SetSpecific("AuthorLetters", f.authorLetters)
	}

	return
}

func (f *CFeature) ProcessGroupsPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	authorGroups := make([]*quote.AuthorsGroup, 0)

	authors := p.Context().Strings("Authors")
	if len(authors) == 0 {
		p.Context().SetSpecific("NumAuthors", f.dbh.TotalAuthors())
		p.Context().SetSpecific("AuthorLetters", f.authorLetters)
		return
	}

	cache := make(map[string]*quote.AuthorsGroup)
	unique := make(map[string]struct{})
	var ok bool
	for _, authorName := range authors {
		var authorKey string
		if authorKey, ok = f.dbh.GetAuthorKeyFrom(authorName); !ok {
			continue
		}
		lastNameKey := quote.GetLastNameKey(authorName)
		if _, present := cache[lastNameKey]; !present {
			cache[lastNameKey] = &quote.AuthorsGroup{
				Key: lastNameKey,
			}
		}
		if _, present := unique[authorKey]; !present {
			cache[lastNameKey].Authors = append(cache[lastNameKey].Authors, &quote.Author{
				Key:  authorKey,
				Name: authorName,
			})
		}
	}

	for _, key := range maps.SortedKeys(cache) {
		group := cache[key]
		sort.Slice(group.Authors, func(i, j int) (less bool) {
			a, b := group.Authors[i], group.Authors[j]
			lna, lnb := clStrings.LastName(a.Name), clStrings.LastName(b.Name)
			if lna == lnb {
				less = natural.Less(a.Name, b.Name)
			} else {
				less = natural.Less(lna, lnb)
			}
			return
		})
		authorGroups = append(authorGroups, group)
	}

	p.Context().SetSpecific("AuthorGroups", authorGroups)

	pg = p
	processed = true
	return
}
