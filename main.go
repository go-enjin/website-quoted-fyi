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

package main

import (
	"fmt"
	"os"

	zipContent "github.com/go-enjin/be/features/fs/zips/content"
	"github.com/go-enjin/be/features/pages/caching/stock-pgc"
	kws "github.com/go-enjin/be/features/pages/indexing/stock-kws"
	pql "github.com/go-enjin/be/features/pages/indexing/stock-pql"
	"github.com/go-enjin/be/pkg/cli/env"
	"github.com/go-enjin/be/pkg/log"
	bePath "github.com/go-enjin/be/pkg/path"

	// "github.com/go-enjin/be/features/pages/indexing/leveldb-indexing"
	"github.com/go-enjin/be/features/pages/query"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/features/requests/headers/proxy"
	"github.com/go-enjin/be/features/restrict/basic-auth"
	"github.com/go-enjin/be/pkg/profiling"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/authors"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/build"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/q"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/quote"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/random"
	qfSearch "github.com/go-enjin/website-quoted-fyi/pkg/features/search"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/topics"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/words"

	"github.com/go-enjin/golang-org-x-text/language"

	"github.com/go-enjin/be/features/pages/robots"
	"github.com/go-enjin/be/pkg/lang"

	semantic "github.com/go-enjin/semantic-enjin-theme"

	"github.com/go-enjin/be"
	"github.com/go-enjin/be/features/log/papertrail"
	"github.com/go-enjin/be/features/pages/formats"
	"github.com/go-enjin/be/pkg/feature"
)

var (
	fContent feature.Feature
	fPublic  feature.Feature
	fMenu    feature.Feature

	hotReload bool
)

func setup(eb *be.EnjinBuilder) *be.EnjinBuilder {
	eb.SiteName("Quoted.Fyi").
		SiteTagLine("This IP for your information.").
		SiteCopyrightName("Go-Enjin").
		SiteCopyrightNotice("© 2022 All rights reserved").
		AddFeature(pgc.New().Make()).
		AddFeature(proxy.New().Enable().Make()).
		AddFeature(formats.New().Defaults().AddFormat(q.New().Make()).Make()).
		AddFeature(query.New().Make()).
		AddFeature(quote.New().Make()).
		AddFeature(build.New().Make()).
		AddFeature(words.New().Make()).
		AddFeature(topics.New().Make()).
		AddFeature(random.New().Make()).
		AddFeature(authors.New().Make()).
		// AddFeature(indexing.New().Make()).
		// AddFeature(fts.New().Make()).
		AddFeature(kws.New().Make()).
		AddFeature(pql.New().Make()).
		AddFeature(search.New().SetPath("/search").Make()).
		AddFeature(qfSearch.New().Make()).
		AddTheme(semantic.SemanticEnjinTheme()).
		AddTheme(quotedFyiTheme()).
		SetTheme("quoted-fyi")
	return eb
}

func features(eb feature.Builder) feature.Builder {
	return eb.
		AddFeature(auth.New().EnableEnv(true).Make()).
		AddFeature(papertrail.Make()).
		AddFeature(robots.New().
			AddRuleGroup(robots.NewRuleGroup().
				AddUserAgent("*").AddAllowed("/").Make(),
			).Make()).
		SetStatusPage(404, "/404").
		SetStatusPage(500, "/500").
		HotReload(hotReload)
}

func main() {
	var quotesZipPath string
	if quotesZipPath = env.Get("QUOTES_ZIP_PATH", ""); quotesZipPath == "" {
		log.FatalF("QUOTES_ZIP_PATH not set")
	} else if !bePath.IsFile(quotesZipPath) {
		log.FatalF("QUOTES_ZIP_PATH not found or not a file: %v", quotesZipPath)
	}
	defer profiling.Stop()
	enjin := be.New()
	setup(enjin).SiteTag("QF").
		SiteDefaultLanguage(language.English).
		SiteLanguageMode(lang.NewPathMode().Make()).
		SiteSupportedLanguages(language.English)
	features(enjin).
		AddFeature(fMenu).
		AddFeature(fPublic).
		AddFeature(fContent).
		AddFeature(zipContent.New().
			MountPathZip("/q/", "quotes", quotesZipPath).
			Make(),
		)
	if err := enjin.Build().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "enjin.Run error: %v\n", err)
		os.Exit(1)
	}
}