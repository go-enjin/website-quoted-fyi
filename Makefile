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

COMMON_TAGS += htmlify
COMMON_TAGS += papertrail
COMMON_TAGS += header_proxy
COMMON_TAGS += basic_auth
COMMON_TAGS += driver_kws
#COMMON_TAGS += driver_kvs_gocache memory memshard imcache bigcache ristretto
COMMON_TAGS += driver_kvs_gocache memory
COMMON_TAGS += page_pql page_query page_search
COMMON_TAGS += page_robots
COMMON_TAGS += driver_fs_embed driver_fs_zip
COMMON_TAGS += fs_theme fs_menu fs_content fs_public

BUILD_TAGS     = prd embeds $(COMMON_TAGS)
DEV_BUILD_TAGS = dev locals $(COMMON_TAGS)

## Custom go.mod locals
GOPKG_KEYS := SET

## Semantic Enjin Theme
SET_GO_PACKAGE := github.com/go-enjin/semantic-enjin-theme
SET_LOCAL_PATH := ../semantic-enjin-theme

## Go-Enjin gotext package
#GOXT_GO_PACKAGE = github.com/go-enjin/golang-org-x-text
#GOXT_LOCAL_PATH = ../golang-org-x-text

export GOGC=15

include ./Enjin.mk

export BE_BUNTDB_PATH ?= ./buntdb-indexing.db
export BE_LEVELDB_PATH ?= ./leveldb-indexing.db

LANGUAGES = en
LOCALES_CATALOG ?= /dev/null

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

buntdb-clean:
	@echo "# cleaning ${BE_BUNTDB_PATH}"
	@rm -rfv ${BE_BUNTDB_PATH}

buntdb-precache: export BE_DEBUG=true
buntdb-precache: export BE_LOG_LEVEL=debug
buntdb-precache: build
	@if [ -f "${BE_BUNTDB_PATH}" ]; then \
		echo "# updating ${BE_BUNTDB_PATH}"; \
	else \
		echo "# creating ${BE_BUNTDB_PATH}"; \
	fi
	@( ./be-quoted-fyi --buntdb-path=${BE_BUNTDB_PATH} buntdb-precache 2>&1 ) \
		| perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'

leveldb-clean:
	@echo "# cleaning ${BE_LEVELDB_PATH}"
	@rm -rfv ${BE_LEVELDB_PATH}

leveldb-precache: export BE_DEBUG=true
leveldb-precache: export BE_LOG_LEVEL=debug
leveldb-precache: build
	@if [ -f "${BE_LEVELDB_PATH}" ]; then \
		echo "# updating ${BE_LEVELDB_PATH}"; \
	else \
		echo "# creating ${BE_LEVELDB_PATH}"; \
	fi
	@( ./be-quoted-fyi --leveldb-path=${BE_LEVELDB_PATH} leveldb-precache 2>&1 ) \
		| perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'
