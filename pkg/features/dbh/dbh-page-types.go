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
	"net/http"

	"github.com/go-enjin/be/pkg/feature"
)

func (f *CFeature) PageTypeNames() (names []string) {
	return []string{
		"page", "quote", "author", "topic",
	}
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {

	// all pages get totals
	p.Context().SetSpecific("NumWords", f.TotalWords())
	p.Context().SetSpecific("NumQuotes", f.TotalQuotes())
	p.Context().SetSpecific("NumTopics", f.TotalTopics())
	p.Context().SetSpecific("NumAuthors", f.TotalAuthors())
	p.Context().SetSpecific("NumFirstWords", f.TotalFirstWords())
	p.Context().SetSpecific("NumSecondWords", f.TotalSecondWords())

	return
}
