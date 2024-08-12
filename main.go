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

package main

import (
	"fmt"
	"os"

	"github.com/spkg/zipfs"

	"github.com/go-corelibs/env"
	clPath "github.com/go-corelibs/path"
	"github.com/go-corelibs/x-text/language"
	"github.com/go-enjin/be"
	"github.com/go-enjin/be/features/fs/content"
	"github.com/go-enjin/be/features/outputs/htmlify"
	"github.com/go-enjin/be/features/pages/robots"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/features/srv/eql"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/lang"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/presets/defaults"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/authors"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/build"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/dbh"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/q"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/random"
	qfSearch "github.com/go-enjin/website-quoted-fyi/pkg/features/search"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/topics"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/words"
)

const (
	gEqlFeature = "eql"
	gKvsFeature = "qf-kvs"
	gKvsName    = "qfyi"
)

var (
	fContent feature.Feature
	fPublic  feature.Feature
	fMenu    feature.Feature
	fThemes  feature.Feature
	fGocache feature.Feature
	fNonces  feature.Feature
)

func main() {

	var quotesZipPath string
	if quotesZipPath = env.String("QUOTES_ZIP_PATH", ""); quotesZipPath == "" {
		log.FatalF("QUOTES_ZIP_PATH not set")
	} else if !clPath.IsFile(quotesZipPath) {
		log.FatalF("QUOTES_ZIP_PATH not found or not a file: %v", quotesZipPath)
	}

	var quotesZipFS *zipfs.FileSystem
	if v, err := zipfs.New(quotesZipPath); err != nil {
		log.FatalF("QUOTES_ZIP_PATH error while opening: %v", err)
	} else {
		quotesZipFS = v
		log.InfoF("QUOTES_ZIP_PATH loaded: %v", quotesZipPath)
	}

	enjin := be.New().
		SiteName("Quoted.Fyi").
		SiteTag("QF").
		SiteTagLine("Quoted for your information.").
		SiteCopyrightName("Quoted.FYI").
		SiteCopyrightNotice("All rights reserved.").
		SiteDefaultLanguage(language.English).
		SiteSupportedLanguages(language.English).
		SiteLanguageMode(lang.NewPathMode().Make()).
		AddPreset(defaults.New().
			OmitTags(htmlify.Tag).
			AddFormats(q.New().Make()).
			Make()).
		AddFeature(fThemes).
		SetPublicAccess(
			feature.NewAction("enjin", "view", "page"),
			feature.NewAction("fs-content", "view", "page"),
			feature.NewAction("fs-content-quotes", "view", "page"),
		).
		AddFeature(
			fGocache,
			fNonces,
			eql.NewTagged(gEqlFeature).
				Including(build.Tag, words.Tag, topics.Tag, authors.Tag, q.Tag).
				Make(),
			dbh.New().Make(),
			fMenu,
			fPublic,
			fContent,
			content.NewTagged("fs-content-quotes").
				MountZipPath("/q/", "quotes", quotesZipFS).
				AddToIndexProviders(gEqlFeature).
				//SetStartupGC(10). // aggressive due to scale of content and resource limitations on production
				Make(),
			q.New().Make(),
			build.New().Make(),
			words.New().Make(),
			topics.New().Make(),
			random.New().Make(),
			authors.New().Make(),
			search.New().SetSearchPath("/search").Make(),
			qfSearch.New().Make(),
			robots.New().
				SiteRobotsHeader("none").
				AddRuleGroup(robots.NewRuleGroup().
					AddUserAgent("*").
					AddDisallowed("/").
					Make()).
				Make(),
		).
		SetStatusPage(404, "/404").
		SetStatusPage(500, "/500").
		HotReload(false)
	if err := enjin.Build().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "enjin.Run error: %v\n", err)
		os.Exit(1)
	}
}
