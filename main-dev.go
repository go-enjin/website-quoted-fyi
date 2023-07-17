//go:build dev || all

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
	"github.com/go-enjin/be/features/fs/content"
	"github.com/go-enjin/be/features/fs/menu"
	"github.com/go-enjin/be/features/fs/public"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/be/pkg/profiling"
	"github.com/go-enjin/be/pkg/theme"
)

func init() {
	// locals environment, early startup debug logging
	// log.Config.LogLevel = log.LevelDebug
	// log.Config.Apply()

	profiling.Start()
	fMenu = menu.New().MountLocalPath("menus", "menus").Make()
	fPublic = public.New().MountLocalPath("/", "public").Make()
	fContent = content.New().
		MountLocalPath("/", "content").
		AddToIndexProviders(gPqlFeature).
		Make()
}

func quotedFyiTheme() (t *theme.Theme) {
	var err error
	if t, err = theme.NewLocal("enjin", "themes/quoted-fyi"); err != nil {
		log.FatalF("error loading local theme: %v", err)
	} else {
		log.DebugF("loaded local theme: %v", t.Name)
	}
	return
}