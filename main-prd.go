//go:build prd

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
	"embed"

	"github.com/go-enjin/be/features/fs/embeds/content"
	"github.com/go-enjin/be/features/fs/embeds/menu"
	"github.com/go-enjin/be/features/fs/embeds/public"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/theme"
)

//go:embed content/**
var contentFsWWW embed.FS

//go:embed public/**
var publicFs embed.FS

//go:embed menus/**
var menuFsWWW embed.FS

//go:embed themes/**
//go:embed themes/*/layouts/_default/**
var themeFs embed.FS

func init() {
	fMenu = menu.New().MountPathFs("menus", "menus", menuFsWWW).Make()
	fPublic = public.New().MountPathFs("/", "public", publicFs).Make()
	fContent = content.New().
		MountPathFs("/", "content", contentFsWWW).
		Make()

	hotReload = false
}

func quotedFyiTheme() (t *theme.Theme) {
	var err error
	if t, err = theme.NewEmbed("themes/quoted-fyi", themeFs); err != nil {
		log.FatalF("error loading embed theme: %v", err)
	} else {
		log.DebugF("loaded embed theme: %v", t.Name)
	}
	return
}