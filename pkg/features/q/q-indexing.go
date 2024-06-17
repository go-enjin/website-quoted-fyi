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

package q

import (
	"github.com/go-corelibs/enjinql"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

func (f *CFeature) AddSources() (sources enjinql.ConfigSources) {
	return enjinql.ConfigSources{
		enjinql.MakeSourceConfig(
			enjinql.PageSource,
			quote.HashSource,
			enjinql.NewStringValue("hash", 8),
		).AddIndex("hash"),
	}
}

func (f *CFeature) AddToSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	if p.Type() != "quote" {
		return // only quotes have things
	}

	if hash := p.Context().String("QuoteHash"); hash != "" {
		_, err = tx.Insert(quote.HashSource, sid, hash)
	} else {
		log.ErrorF("missing .QuoteHash: %q", p.Url())
	}

	return
}

func (f *CFeature) RemoveFromSource(tx enjinql.SqlTX, sid int64, stub *feature.PageStub, p feature.Page) (err error) {
	// nop, qf site is read-only
	return
}
