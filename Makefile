#!/usr/bin/make --no-print-directory --jobs=1 --environment-overrides -f

# Copyright (c) 2022  The Go-Enjin Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

-include .env

BE_LOCAL_PATH ?= ../be

APP_NAME    := be-quoted-fyi
APP_SUMMARY := quoted.fyi

DENY_DURATION := 600

ADD_TAGS_DEFAULTS := true

COMMON_TAGS += papertrail
COMMON_TAGS += user_auth_basic
COMMON_TAGS += user_base_htenv
COMMON_TAGS += drivers_db gorm sqlite
COMMON_TAGS += driver_kws
COMMON_TAGS += driver_kvs_gocache memory imcache redis
COMMON_TAGS += page_pql
COMMON_TAGS += page_search
COMMON_TAGS += page_robots
COMMON_TAGS += driver_fs_embed
COMMON_TAGS += driver_fs_zip
COMMON_TAGS += srv_eql
COMMON_TAGS += fs_theme fs_menu fs_content fs_public

BUILD_TAGS     = prd embeds $(COMMON_TAGS)
DEV_BUILD_TAGS = dev locals $(COMMON_TAGS)

AUTO_CORELIBS_KEYS := true

## Custom go.mod locals
GOPKG_KEYS += _SEMANTIC_THEME

LANGUAGES := en
LOCALES_CATALOG := /dev/null

#: decrease go gc threshold for production
export GOGC=50

include ./Enjin.mk


gen-theme-locales:
	@echo "# generating quoted-fyi theme locales"
	@${CMD} enjenv be-update-locales \
		-lang=${LANGUAGES} \
		-out=./themes/quoted-fyi/locales \
		./themes/quoted-fyi/layouts \
		./content

gen-locales: BE_PKG_LIST=$(shell enjenv be-pkg-list)
gen-locales: gen-theme-locales
	@echo "# generating locales"
	@${CMD} \
		GOFLAGS="-tags=all" \
		gotext -srclang=en update \
			-lang=${LANGUAGES} \
			-out=${LOCALES_CATALOG} \
				${BE_PKG_LIST} \
				github.com/go-enjin/website-quoted-fyi
	@if [ -d locales ]; then \
		find locales -type f -name "*.gotext.json" -print0 | xargs -n 1 -0 sha256sum; \
	else \
		echo "# error: locales directory not found" 1>&2; \
		false; \
	fi
