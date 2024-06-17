//go:build prd

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
	"embed"

	"github.com/go-enjin/be/drivers/kvs/gocache"
	"github.com/go-enjin/be/features/fs/content"
	"github.com/go-enjin/be/features/fs/menu"
	"github.com/go-enjin/be/features/fs/public"
	"github.com/go-enjin/be/features/fs/themes"
	"github.com/go-enjin/be/features/srv/factories/nonces"
	semantic "github.com/go-enjin/semantic-enjin-theme"
)

//go:embed content/**
var contentFsWWW embed.FS

//go:embed public/**
var publicFs embed.FS

//go:embed menus/**
var menuFsWWW embed.FS

//go:embed themes/**
var themeFs embed.FS

func init() {
	fMenu = menu.New().MountEmbedPath("menus", "menus", menuFsWWW).Make()
	fPublic = public.New().MountEmbedPath("/", "public", publicFs).Make()
	fContent = content.New().
		MountEmbedPath("/", "content", contentFsWWW).
		AddToIndexProviders(gEqlFeature).
		Make()
	fThemes = themes.New().
		Include(semantic.Theme()).
		EmbedTheme("themes/quoted-fyi", themeFs).
		SetTheme("quoted-fyi").
		Make()

	fGocache = gocache.NewTagged(gKvsFeature).AddRedisCache(gKvsName).Make()
	fNonces = nonces.New().SetKeyValueCache(gKvsFeature, gKvsName).Make()
}
