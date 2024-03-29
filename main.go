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

	"github.com/go-enjin/be"
	"github.com/go-enjin/be/drivers/kvs/gocache"
	"github.com/go-enjin/be/drivers/kws"
	"github.com/go-enjin/be/features/fs/content"
	"github.com/go-enjin/be/features/outputs/htmlify"
	"github.com/go-enjin/be/features/pages/pql"
	"github.com/go-enjin/be/features/pages/robots"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/pkg/cli/env"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/lang"
	"github.com/go-enjin/be/pkg/log"
	bePath "github.com/go-enjin/be/pkg/path"
	"github.com/go-enjin/be/pkg/profiling"
	"github.com/go-enjin/be/presets/defaults"
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
	fThemes  feature.Feature
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
		SiteTag("QF").
		SiteDefaultLanguage(language.English).
		SiteSupportedLanguages(language.English).
		SiteLanguageMode(lang.NewPathMode().Make()).
		AddPreset(defaults.New().
			Prepend(
				quote.New().Make(),
				build.NewTagged(gBuildQuotesFeature).SetKeywordProvider(gKwsFeature).Make(),
				words.New().SetKeywordProvider(gKwsFeature).Make(),
				topics.New().Make(),
				random.New().SetKeywordProvider(gKwsFeature).Make(),
				authors.NewTagged(gAuthorsFeature).Make(),
				search.New().SetSearchPath("/search").Make(),
				qfSearch.New().Make(),
			).
			OmitTags(htmlify.Tag).
			AddFormats(q.New().Make()).
			Make()).
		AddFeature(fThemes).
		SetPublicAccess(
			feature.NewAction("enjin", "view", "page"),
			feature.NewAction("fs-content", "view", "page"),
			feature.NewAction("fs-content-quotes", "view", "page"),
		).
		AddFeature(gocache.NewTagged(gPqlKwsFeature).AddMemoryCache(gPqlKwsCache).Make()).
		AddFeature(gocache.NewTagged(gKvsFeature).AddMemoryCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddIMCacheCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddIMCacheCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddMemShardCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddMemShardCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddBigCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddBigCache(gKvsCache).Make()).
		//AddFeature(gocache.NewTagged(gPqlKwsFeature).AddRistrettoCache(gPqlKwsCache).Make()).
		//AddFeature(gocache.NewTagged(gKvsFeature).AddRistrettoCache(gKvsCache).Make()).
		AddFeature(pql.NewTagged(gPqlFeature).
			IncludeContextKeys("QuoteAuthor", "QuoteAuthorKey", "QuoteCategories", "QuoteCategoryKeys", "QuoteShasum", "QuoteHash").
			SetKeyValueCache(gPqlKwsFeature, gPqlKwsCache).
			Make()).
		AddFeature(kws.NewTagged(gKwsFeature).Make()).
		//AddFeature(quote.New().Make()).
		//AddFeature(build.NewTagged(gBuildQuotesFeature).SetKeywordProvider(gKwsFeature).Make()).
		//AddFeature(words.New().SetKeywordProvider(gKwsFeature).Make()).
		//AddFeature(topics.New().Make()).
		//AddFeature(random.New().SetKeywordProvider(gKwsFeature).Make()).
		//AddFeature(authors.NewTagged(gAuthorsFeature).Make()).
		//AddFeature(search.New().SetSearchPath("/search").Make()).
		//AddFeature(qfSearch.New().Make()).
		AddFeature(robots.New().
			AddRuleGroup(robots.NewRuleGroup().
				AddUserAgent("*").
				AddAllowed("/").
				Make(),
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
			SetStartupGC(25). // aggressive due to scale of content and resource limitations on production
			Make(),
		)
	if err := enjin.Build().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "enjin.Run error: %v\n", err)
		os.Exit(1)
	}
}