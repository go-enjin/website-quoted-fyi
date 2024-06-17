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

package build

import (
	"net/http"

	"github.com/go-enjin/be/pkg/feature"
)

func (f *CFeature) PageTypeNames() (names []string) {
	names = append(names, "build", "builder", "building")
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {

	switch p.Type() {
	case "build":
		pg, redirect, processed, err = f.ProcessBuildPageType(r, p)
	case "builder":
		pg, redirect, processed, err = f.ProcessBuilderPageType(r, p)
	case "building":
		pg, redirect, processed, err = f.ProcessBuildingPageType(r, p)
	}

	return
}

func (f *CFeature) ProcessBuildPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit build page type: %v", p.Url())
	p.Context().SetSpecific("FirstWordFirstLetters", f.dbh.GetFirstWordLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) ProcessBuilderPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit builder page type: %v", p.Url())
	p.Context().SetSpecific("FirstWordFirstLetters", f.dbh.GetFirstWordLetters())
	pg = p
	processed = true
	return
}

func (f *CFeature) ProcessBuildingPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit building page type: %v", p.Url())
	p.Context().SetSpecific("FirstWordFirstLetters", f.dbh.GetFirstWordLetters())
	pg = p
	processed = true
	return
}
