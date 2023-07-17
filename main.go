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

	"github.com/spkg/zipfs"

	"github.com/go-enjin/golang-org-x-text/language"

	semantic "github.com/go-enjin/semantic-enjin-theme"

	"github.com/go-enjin/be"
	"github.com/go-enjin/be/drivers/kvs/gocache"
	"github.com/go-enjin/be/drivers/kws"
	"github.com/go-enjin/be/features/fs/content"
	"github.com/go-enjin/be/features/log/papertrail"
	"github.com/go-enjin/be/features/pages/formats"
	"github.com/go-enjin/be/features/pages/pql"
	"github.com/go-enjin/be/features/pages/query"
	"github.com/go-enjin/be/features/pages/robots"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/features/requests/headers/proxy"
	"github.com/go-enjin/be/features/user/auth/basic"
	"github.com/go-enjin/be/features/user/base/htenv"
	"github.com/go-enjin/be/pkg/cli/env"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/lang"
	"github.com/go-enjin/be/pkg/log"
	bePath "github.com/go-enjin/be/pkg/path"
	"github.com/go-enjin/be/pkg/profiling"
	"github.com/go-enjin/be/pkg/userbase"

	"github.com/go-enjin/website-quoted-fyi/pkg/features/authors"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/build"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/q"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/quote"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/random"
	qfSearch "github.com/go-enjin/website-quoted-fyi/pkg/features/search"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/topics"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/words"
)

const (
	gPqlKwsFeature      = "pql-kws-feature"
	gPqlKwsCache        = "pql-kws-cache"
	gPqlFeature         = "pql-feature"
	gKvsFeature         = "kvs-feature"
	gKvsCache           = "kvs-cache"
	gKwsFeature         = "kws-feature"
	gBuildQuotesFeature = "build-quotes-feature"
	gAuthorsFeature     = "authors-feature"
)

var (
	fContent feature.Feature
	fPublic  feature.Feature
	fMenu    feature.Feature
)

func main() {
	var quotesZipPath string
	if quotesZipPath = env.Get("QUOTES_ZIP_PATH", ""); quotesZipPath == "" {
		log.FatalF("QUOTES_ZIP_PATH not set")
	} else if !bePath.IsFile(quotesZipPath) {
		log.FatalF("QUOTES_ZIP_PATH not found or not a file: %v", quotesZipPath)
	}

	var quotesZipFS *zipfs.FileSystem
	if v, err := zipfs.New(quotesZipPath); err != nil {
		log.FatalF("QUOTES_ZIP_PATH error while opening: %v", err)
	} else {
		quotesZipFS = v
		log.InfoF("QUOTES_ZIP_PATH loaded: %v", quotesZipPath)
	}

	defer profiling.Stop()
	enjin := be.New().
		SiteName("Quoted.Fyi").
		SiteTagLine("Quoted for your information.").
		SiteCopyrightName("Go-Enjin").
		SiteCopyrightNotice("© 2023 All rights reserved").
		AddTheme(semantic.Theme()).
		AddTheme(quotedFyiTheme()).
		SetTheme("quoted-fyi").
		SiteTag("QF").
		SiteDefaultLanguage(language.English).
		SiteSupportedLanguages(language.English).
		SiteLanguageMode(lang.NewPathMode().Make()).
		SetPublicAccess(
			userbase.NewAction("enjin", "view", "page"),
			userbase.NewAction("fs-content", "view", "page"),
			userbase.NewAction("fs-content-quotes", "view", "page"),
		).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddMemoryCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddMemoryCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddRistrettoCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddRistrettoCache(gKvsCache).Make()).
		AddFeature(gocache.NewTagged(gPqlKwsFeature).AddMemShardCache(gPqlKwsCache).Make()).
		AddFeature(gocache.NewTagged(gKvsFeature).AddMemShardCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddBigCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddBigCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddIMCacheCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddIMCacheCache(gKvsCache).Make()).
		AddFeature(pql.NewTagged(gPqlFeature).
			IncludeContextKeys("QuoteAuthor", "QuoteAuthorKey", "QuoteCategories", "QuoteCategoryKeys", "QuoteShasum", "QuoteHash").
			SetKeyValueCache(gPqlKwsFeature, gPqlKwsCache).
			Make()).
		AddFeature(kws.NewTagged(gKwsFeature).Make()).
		AddFeature(proxy.New().Enable().Make()).
		AddFeature(formats.New().Defaults().AddFormat(q.New().Make()).Make()).
		AddFeature(quote.New().Make()).
		AddFeature(build.NewTagged(gBuildQuotesFeature).Make()).
		AddFeature(words.New().Make()).
		AddFeature(topics.New().Make()).
		AddFeature(random.New().Make()).
		AddFeature(authors.NewTagged(gAuthorsFeature).Make()).
		AddFeature(search.New().SetSearchPath("/search").Make()).
		AddFeature(qfSearch.New().Make()).
		AddFeature(query.New().Make()).
		AddFeature(htenv.NewTagged("htenv").Make()).
		AddFeature(basic.NewTagged("basic-auth").AddUserbase("htenv", "htenv", "htenv").Make()).
		AddFeature(papertrail.Make()).
		AddFeature(robots.New().
			AddRuleGroup(robots.NewRuleGroup().
				AddUserAgent("*").AddAllowed("/").Make(),
			).Make()).
		SetStatusPage(404, "/404").
		SetStatusPage(500, "/500").
		HotReload(false).
		AddFeature(fMenu).
		AddFeature(fPublic).
		AddFeature(fContent).
		AddFeature(content.NewTagged("fs-content-quotes").
			MountZipPath("/q/", "quotes", quotesZipFS).
			//MountLocalPath("/q/", "quotes").
			AddToIndexProviders(gPqlFeature, gBuildQuotesFeature, gAuthorsFeature).
			AddToSearchProviders(gKwsFeature).
			Make(),
		)
	if err := enjin.Build().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "enjin.Run error: %v\n", err)
		os.Exit(1)
	}
}