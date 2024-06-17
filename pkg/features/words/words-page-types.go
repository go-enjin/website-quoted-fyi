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

package words

import (
	"net/http"

	"github.com/go-enjin/be/pkg/feature"
)

func (f *CFeature) PageTypeNames() (names []string) {
	names = append(names, "words", "word")
	return
}

func (f *CFeature) ProcessRequestPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// reqArgv := site.GetRequestArgv(r)

	switch p.Type() {
	case "words":
		pg, redirect, processed, err = f.ProcessGroupsPageType(r, p)
	case "word":
		pg, redirect, processed, err = f.ProcessSinglePageType(r, p)
	}

	return
}

func (f *CFeature) ProcessSinglePageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit word page type: %v", p.Url())
	return
}

func (f *CFeature) ProcessGroupsPageType(r *http.Request, p feature.Page) (pg feature.Page, redirect string, processed bool, err error) {
	// log.WarnF("hit words group type: %v", p.Url())
	return
}
