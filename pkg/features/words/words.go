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
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/urfave/cli/v2"

	"github.com/go-corelibs/rxp"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/dbh"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "quote-word-pages"

type Feature interface {
	feature.Feature
	feature.UseMiddleware
	feature.PageTypeProcessor
	feature.QueryIndexSourceFeature
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature

	dbh   dbh.Feature
	theme feature.Theme

	sync.RWMutex
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = Tag
	f.CFeature.Construct(f)
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
	if t, err := f.Enjin.GetTheme(); err != nil {
		log.FatalF("error getting enjin theme: %v", err)
	} else {
		f.theme = t
	}
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	err = f.CFeature.Startup(ctx)
	return
}

func (f *CFeature) PostStartup(ctx *cli.Context) (err error) {
	if tfs := feature.FilterTyped[dbh.Feature](f.Enjin.Features().List()); len(tfs) > 0 {
		f.dbh = tfs[0]
	} else {
		err = fmt.Errorf("a dbh.Feature is required")
		return
	}
	gStartupCache.uniqueSWID.Close()
	gStartupCache.uniqueFWID.Close()
	gStartupCache.lookupWORD.Close()
	gStartupCache.lookupWID.Close()
	gStartupCache = nil
	return
}

// ^/w/([^/]+)/??$
var rxPagePath = rxp.Pattern{}.
	Caret().
	Text("/w/").
	Not(rxp.Text("/"), "+", "c").
	Text("/", "?").
	Dollar()

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote words middleware")
	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if unescaped, err := url.PathUnescape(path); err == nil {
				path = unescaped
			}

			switch {
			case rxPagePath.MatchString(path):
				m := rxPagePath.FindAllStringSubmatch(path, 1)
				pathWord := m[0][1]
				f.ProcessPagePath(pathWord, w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
