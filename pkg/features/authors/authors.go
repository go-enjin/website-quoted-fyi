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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/maruel/natural"
	"github.com/urfave/cli/v2"

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

const Tag feature.Tag = "quote-author-pages"

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

	dbh dbh.Feature
	eql feature.QueryIndexFeature

	authorLetters []string
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

	// list of author letters, sorted naturally
	if _, results, ee := f.eql.PerformLookup(`LOOKUP DISTINCT author.Letter`); ee != nil || results.Len() == 0 {
		err = fmt.Errorf("error getting list of author letters: %w", ee)
		return
	} else {
		for _, result := range results {
			if letter := result.String("letter", ""); letter != "" {
				f.authorLetters = append(f.authorLetters, letter)
			}
		}
		sort.Sort(natural.StringSlice(f.authorLetters))
	}

	log.InfoF("found author letters: %d, total: %d", len(f.authorLetters), f.dbh.TotalAuthors())
	gStartupCache.Close()
	gStartupCache = nil
	return
}

var (
	// ^/a/([^/]+)/??$
	rxPagePath = rxp.Pattern{}.
			Caret().
			Text("/a/").
			Not(rxp.Text("/"), "+", "c").
			Text("/", "??").
			Dollar()

	// ^/authors/([a-zA-Z0-9])?/??
	rxGroupPath = rxp.Pattern{}.
			Caret().
			Text("/authors/").
			Alnum("?", "c").
			Text("/", "??").
			Dollar()
)

func (f *CFeature) Use(s feature.System) feature.MiddlewareFn {
	log.DebugF("including quote authors middleware")
	return func(next http.Handler) (this http.Handler) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if unescaped, err := url.PathUnescape(path); err == nil {
				path = unescaped
			}

			switch path {
			case "/a", "/a/":
				reqArgv := argv.Get(r)
				reqArgv.Path = "/authors/"
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				return
			}

			switch {

			case rxPagePath.MatchString(path):
				if m := rxPagePath.FindAllStringSubmatch(path, 1); len(m[0]) == 2 {
					authorKey := strings.ToLower(m[0][1])
					f.ProcessPagePath(authorKey, w, r)
					return
				}
				reqArgv := argv.Get(r)
				reqArgv.Path = "/authors/"
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				return

			case rxGroupPath.MatchString(path):
				if m := rxGroupPath.FindAllStringSubmatch(path, 1); len(m[0]) == 2 {
					groupChar := strings.ToLower(m[0][1])
					f.ProcessGroupPath(groupChar, w, r)
					return
				}
				reqArgv := argv.Get(r)
				reqArgv.Path = "/authors/"
				f.Enjin.ServeRedirect(reqArgv.String(), w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
