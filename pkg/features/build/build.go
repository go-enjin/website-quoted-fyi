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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/go-corelibs/context"
	"github.com/go-corelibs/rxp"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/request/argv"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/dbh"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "build-quote-pages"

type Feature interface {
	feature.Feature
	feature.UseMiddleware
	feature.PageTypeProcessor
	feature.PageContextModifier
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature

	dbh dbh.Feature
	eql feature.QueryIndexFeature

	firstWordFirstLetters []string
}

func New() MakeFeature {
	return NewTagged(Tag)
}

func NewTagged(tag feature.Tag) MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = tag
	f.CFeature.Construct(f)
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Build(b feature.Buildable) (err error) {
	if err = f.CFeature.Build(b); err != nil {
		return
	}
	return
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	if err = f.CFeature.Startup(ctx); err != nil {
		return
	}

	if found := f.Enjin.GetQueryIndexFeatures(); len(found) > 0 {
		f.eql = found[0]
	} else {
		err = fmt.Errorf("%v feature requires at least one feature.QueryIndexFeature", f.Tag())
		return
	}

	if f.dbh = feature.FirstTyped[dbh.Feature](f.Enjin.Features().List()); f.dbh == nil {
		err = fmt.Errorf("%v features requires dbh.Feature", f.Tag())
		return
	}

	return
}

func (f *CFeature) PostStartup(ctx *cli.Context) (err error) {
	return
}

func (f *CFeature) FilterPageContext(themeCtx, pageCtx context.Context, r *http.Request) (themeOut context.Context) {
	themeOut = themeCtx
	if pgType, ok := pageCtx.Get("Type").(string); ok && pgType == "quote" {
		var shortest string
		if builderKey, ok := f.dbh.GetBuilderKeyFor(pageCtx.String("Shasum", "")); ok {
			shortest, _ = f.dbh.GetShortestBuilderKey(builderKey)
		}
		themeOut.SetSpecific("QuoteBuilderKey", f.dbh.GetHumanBuilderKey(shortest))
	}
	return
}

var (
	// ^/b/([^/]+)/??$
	rxPagePath = rxp.Pattern{}.
			Caret().
			Text("/b/").
			Not(rxp.Text("/"), "+", "c").
			Text("/", "??").
			Dollar()

	// ^/build/([a-zA-Z0-9])?/??
	rxGroupPath = rxp.Pattern{}.
			Caret().
			Text("/build/").
			Alnum("?", "c").
			Text("/", "??").
			Dollar()
)

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote words middleware")

	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if unescaped, err := url.PathUnescape(path); err == nil {
				path = unescaped
			}

			switch {
			case path == "/b" || path == "/b/":
				reqArgv := argv.Get(r)
				reqArgv.Path = "/build/"
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				return

			case rxPagePath.MatchString(path):
				if m := rxPagePath.FindAllStringSubmatch(path, 1); len(m[0]) == 2 {
					requestedPath := m[0][1]
					f.ProcessPagePath(requestedPath, w, r)
					return
				}
				reqArgv := argv.Get(r)
				reqArgv.Path = "/build/"
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				return

			case rxGroupPath.MatchString(path):
				if m := rxGroupPath.FindAllStringSubmatch(path, 1); len(m[0]) == 2 {
					groupChar := strings.ToLower(m[0][1])
					f.ProcessGroupPath(groupChar, w, r)
					return
				}
				reqArgv := argv.Get(r)
				reqArgv.Path = "/build/"
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
