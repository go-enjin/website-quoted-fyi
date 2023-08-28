#!/usr/bin/make --no-print-directory --jobs=1 --environment-overrides -f

# Copyright (c) 2023  The Go-Enjin Authors
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

#: uncomment to echo instead of execute
#CMD=echo

.PHONY: audit audit-% be-update build build-dev-run check clean
.PHONY: con-% dev dist-clean help local
.PHONY: profile.cpu profile.mem release release-dev-run run stop tidy unlocal
.PHONY: yarn-% _audit _enjenv _golang _nodejs
.PHONY: _check_make_target _clean _enjenv_bin
.PHONY: _enjenv_present _env_run_vars _golang_nancy_installed
.PHONY: _has_feature _is_nodejs_tag
.PHONY: _make_go_local _make_go_unlocal
.PHONY: _run _upx_build
.PHONY: _validate_extra_pkgs _yarn_run _yarn_run_script
.PHONY: _yarn_tag_install
.PHONY: list-make-targets

ENJIN_MK_VERSION := v0.2.11

SHELL = /bin/bash

ifeq ($(origin APP_NAME),undefined)
APP_NAME := $(shell basename `pwd`)
endif
APP_SUMMARY ?= Go-Enjin
ENV_PREFIX  ?= BE
APP_PREFIX  ?= ${USER}

UNTAGGED_VERSION ?= v0.0.0

BE_DEBUG ?= false

LISTEN        ?=
PORT          ?= 3334
DEBUG         ?= ${BE_DEBUG}
STRICT        ?= false
DOMAIN        ?=
DENY_DURATION ?= 86400

BUILD_TAGS        ?=
BUILD_ARGV        ?=
BUILD_LDFLAGS     ?=
DEV_BUILD_TAGS    ?= ${BUILD_TAGS}
DEV_BUILD_ARGV    ?=
DEV_BUILD_GCFLAGS ?=

PRESET_TAGS ?=

ifeq (${ADD_TAGS_ESSENTIALS},true)
PRESET_TAGS += requests_deny
PRESET_TAGS += htmlify
PRESET_TAGS += header_proxy
PRESET_TAGS += page_query
PRESET_TAGS += page_partials
PRESET_TAGS += page_funcmaps
PRESET_TAGS += srv_pages
PRESET_TAGS += srv_listener_httpd
endif

ifeq (${ADD_TAGS_DEFAULTS},true)
PRESET_TAGS += requests_deny
PRESET_TAGS += user_base_htenv
PRESET_TAGS += user_auth_basic
PRESET_TAGS += htmlify
PRESET_TAGS += header_proxy
PRESET_TAGS += page_query
PRESET_TAGS += page_partials
PRESET_TAGS += page_funcmaps
PRESET_TAGS += page_permalink
PRESET_TAGS += srv_pages
PRESET_TAGS += srv_listener_httpd
endif

ifneq (${PRESET_TAGS},)
BUILD_TAGS     += ${PRESET_TAGS}
DEV_BUILD_TAGS += ${PRESET_TAGS}
endif

GOPKG_KEYS ?=

CLEAN      ?= ${APP_NAME} cpu.pprof mem.pprof
DIST_CLEAN ?=
EXTRA_CLEAN ?=

EXTRA_BUILD_TARGET_DEPS ?=

GOLANG ?= 1.20.1
GO_MOD ?= 1020
NODEJS ?=

RELEASE_BUILD ?= false
PRE_RELEASE_BUILD ?= false

GO_ENJIN_PKG ?= github.com/go-enjin/be

BE_PATH       ?= ../be
BE_LOCAL_PATH ?= ${BE_PATH}

_INTERNAL_BUILD_LOG_ := /dev/null
#_INTERNAL_BUILD_LOG_ := ./build.log

DEFAULT_CONSOLE_KEY ?=
BE_CONSOLE_KEYS ?=

HEROKU_BIN := $(shell which heroku)

DLV_PORT ?= 2345
DLV_DEBUG ?=

DLV_BIN := $(shell which dlv)

ENJENV_PKG  ?= github.com/go-enjin/enjenv/cmd/enjenv@latest
ENJENV_DIR_NAME ?= .enjenv
ENJENV_DIR ?= ${ENJENV_DIR_NAME}

PROFILE_PATH ?= .

RUN_ARGV ?=

#
#: dynamically defined global variables and functions
#

ifeq ($(origin ENJENV_BIN),undefined)
ENJENV_BIN:=$(shell which enjenv)
endif
ifeq ($(origin ENJENV_EXE),undefined)
ENJENV_EXE:=$(shell \
	echo "ENJENV_EXE" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ "${ENJENV_BIN}" != "" -a -x "${ENJENV_BIN}" ]; then \
		echo "${ENJENV_BIN}"; \
	else \
		if [ -x "${PWD}/.enjenv.bin" ]; then \
			echo "${PWD}/.enjenv.bin"; \
		elif [ -d "${PWD}/.bin" -a -x "${PWD}/.bin/enjenv" ]; then \
			echo "${PWD}/.bin/enjenv"; \
		else \
			echo "ERROR"; \
		fi; \
	fi)
endif

ifeq ($(origin ENJENV_PATH),undefined)
ENJENV_PATH := $(shell \
	echo "_enjenv_path" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -x "${ENJENV_EXE}" ]; then \
		${ENJENV_EXE}; \
	elif [ -d "${PWD}/${ENJENV_DIR}" ]; then \
		echo "${PWD}/${ENJENV_DIR}"; \
	fi)
endif

ifeq ($(origin VERSION),undefined)
VERSION := $(shell \
	echo "_be_version" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -x "${ENJENV_EXE}" ]; then \
		${ENJENV_EXE} git-tag --untagged ${UNTAGGED_VERSION}; \
	fi)
endif
ifeq ($(origin RELEASE),undefined)
RELEASE := $(shell \
	echo "_be_release" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -x "${ENJENV_EXE}" ]; then \
		${ENJENV_EXE} rel-ver; \
	fi)
endif

_MAKE_TARGETS := $($(MAKE) list-make-targets 2>/dev/null)

_ALL_FEATURES_PRESENT := $(shell ${ENJENV_EXE} features list 2>/dev/null)

_LIST_PACKAGE_JSON := $(shell \
	echo "_list_package_json" >> ${_INTERNAL_BUILD_LOG_}; \
	ls */package.json 2> /dev/null)

_LIST_NODE_PATHS := $(shell \
	echo "_list_node_paths" >> ${_INTERNAL_BUILD_LOG_}; \
	ls */package.json 2> /dev/null | while read P; do \
		dirname $${P}; \
	done)

_ENJENV_PRESENT := $(shell \
	echo "_enjenv_present" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -n "${ENJENV_EXE}" -a -x "${ENJENV_EXE}" ]; then \
		echo "present"; \
	fi)

_GO_PRESENT := $(shell \
	echo "_go_present" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -n "$(call _has_feature,go)" ]; then \
		echo "present"; \
	fi)

_YARN_PRESENT := $(shell \
	echo "_yarn_present" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -n "$(call _has_feature,yarn)" ]; then \
		echo "present"; \
	fi)

_DEPS_PRESENT := $(shell \
	echo "_deps_present" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ \
			"${_ENJENV_PRESENT}"  == "present" \
			-a "${_GO_PRESENT}"   == "present" \
			-a "${_YARN_PRESENT}" == "present" \
	]; then \
		echo "deps-present"; \
	fi)

_BUILD_ARGS = $(shell \
	echo "_build_args" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ "${RELEASE_BUILD}" == "true" ]; then \
		echo " --optimize"; \
	fi)

_BUILD_ARGV = $(shell \
	echo "_build_argv" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ "${RELEASE_BUILD}" == "true" -o "${PRE_RELEASE_BUILD}" == "true" ]; then \
		echo "${BUILD_ARGV}"; \
	else \
		echo "${DEV_BUILD_ARGV}"; \
	fi)

_EXTRA_LDFLAGS := $(shell \
	if [ "${RELEASE_BUILD}" == true ]; then \
		echo "${BUILD_LDFLAGS}"; \
	else \
		echo "${DEV_BUILD_LDFLAGS}"; \
	fi)

_EXTRA_GCFLAGS := $(shell \
	if [ "${RELEASE_BUILD}" == true ]; then \
		echo "${BUILD_GCFLAGS}"; \
	else \
		echo "${DEV_BUILD_GCFLAGS}"; \
	fi)

_BUILD_TAGS = $(shell \
echo "_build_tags" >> ${_INTERNAL_BUILD_LOG_}; \
if [ "${RELEASE_BUILD}" == "true" -o "${PRE_RELEASE_BUILD}" == "true" ]; then \
	if [ "${BUILD_TAGS}" != "" ]; then \
		tags=`echo "${BUILD_TAGS}" | perl -pe 's!\s+!,!msg;s!,$$!!;'`; \
		echo "-tags $${tags}"; \
	fi; \
elif [ "${DEV_BUILD_TAGS}" != "" ]; then \
	tags=`echo "${DEV_BUILD_TAGS}" | perl -pe 's!\s+!,!msg;s!,$$!!;'`; \
	echo "-tags $${tags}"; \
fi)

define _check_make_target
	echo "_check_make_target $(1)" >> ${_INTERNAL_BUILD_LOG_}; \
	if echo "${_MAKE_TARGETS}" | egrep -q "^$(1)\$"; then \
		echo "false"; \
	else \
		echo "true"; \
	fi
endef

define _clean
	echo "_clean $(1)" >> ${_INTERNAL_BUILD_LOG_}; \
	for thing in $(1); do \
		if [ -d "$${thing}" ]; then \
			rm -rf "$${thing}" && \
				echo "removed: \"$${thing}\" (recursively)"; \
		elif [ -f "$${thing}" ]; then \
			rm -f "$${thing}" && \
				echo "removed: \"$${thing}\""; \
		fi; \
	done
endef

define _upx_build
	if [ -n "$(1)" ]; then \
		echo "_upx_build" >> ${_INTERNAL_BUILD_LOG_}; \
		if [ -x /usr/bin/upx ]; then \
			echo -n "# packing: $(1) - "; \
			du -hs "$(1)" | awk '{print $$1}'; \
			/usr/bin/upx -qq -7 --no-color --no-progress "$(1)"; \
			echo -n "# packed: $(1) - "; \
			du -hs "$(1)" | awk '{print $$1}'; \
			sha256sum "$(1)"; \
		else \
			echo "# upx command not found, skipping binary packing stage"; \
		fi; \
	fi
endef

define _validate_extra_pkgs
$(if ${GOPKG_KEYS},$(foreach key,${GOPKG_KEYS},$(shell \
		if [ \
			-z "$($(key)_GO_PACKAGE)" \
			-o -z "$($(key)_LOCAL_PATH)" \
			-o ! -d "$($(key)_LOCAL_PATH)" \
		]; then \
			echo "echo \"# $(key)_GO_PACKAGE and/or $(key)_LOCAL_PATH not found\"; false;"; \
		fi \
)))
endef

define _make_go_local
echo "_make_go_local $(1) $(2)" >> ${_INTERNAL_BUILD_LOG_}; \
echo "# go.mod local: $(1)"; \
${CMD} ${ENJENV_EXE} go-local "$(1)" "$(2)"
endef

define _make_go_unlocal
echo "_make_go_unlocal $(1)" >> ${_INTERNAL_BUILD_LOG_}; \
echo "# go.mod unlocal $(1)"; \
${CMD} ${ENJENV_EXE} go-unlocal "$(1)"
endef

define _make_extra_pkgs
$(if ${GOPKG_KEYS},$(foreach key,${GOPKG_KEYS},$($(key)_GO_PACKAGE)@latest))
endef

define _make_console_names
$(if ${BE_CONSOLE_KEYS},$(foreach key,${BE_CONSOLE_KEYS},$(shell echo "$($(key)_CONSOLE_NAME)")))
endef

define _first_console_name
$(shell \
	for name in $(call _make_console_names); do \
		echo "$${name}"; \
		break; \
	done)
endef

define _default_console_name
$(shell \
	if [ -n "${DEFAULT_CONSOLE_KEY}" ]; then \
		echo "$($(DEFAULT_CONSOLE_KEY)_CONSOLE_NAME)"; \
	elif [ -n "${BE_CONSOLE_KEYS}" ]; then \
		echo "$(call _first_console_name)"; \
	fi)
endef

define _has_feature
$(shell \
	if [ -n "$(1)" -a "$(1)" != "yarn--" -a "$(1)" != "yarn---install" ]; then \
		for feature in ${_ALL_FEATURES_PRESENT}; do \
			if [ "$${feature}" == "$(1)" ]; then \
				echo "_has_feature $(1) (found)" >> ${_INTERNAL_BUILD_LOG_}; \
				echo "$${feature}"; \
				break; \
			else \
				echo "_has_feature $(1) (is not $${feature})" >> ${_INTERNAL_BUILD_LOG_}; \
			fi; \
		done; \
	fi)
endef

define _env_run_vars
	${ENV_PREFIX}_DEBUG="${DEBUG}" \
	${ENV_PREFIX}_LISTEN="${LISTEN}" \
	${ENV_PREFIX}_PORT="${PORT}" \
	${ENV_PREFIX}_PREFIX="${APP_PREFIX}" \
	${ENV_PREFIX}_STRICT="${STRICT}" \
	${ENV_PREFIX}_DOMAIN="${DOMAIN}" \
	${ENV_PREFIX}_DENY_DURATION="${DENY_DURATION}"
endef

ifdef override_env_run_vars
define _get_run_vars
	$(call override_env_run_vars)
endef
else
define _get_run_vars
	$(call _env_run_vars)
endef
endif

define _is_nodejs_tag
	echo "_is_nodejs_tag $(1)" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ "$(1)" != "" -a -d "$(1)" ]; then \
		if [ ! -f "$(1)/package.json" ]; then \
			echo "# $(1)/package.json not found"; \
			false; \
		fi; \
	else \
		echo "# \"$(1)/package.json\" not found"; \
		false; \
	fi
endef

define _yarn_tag_install
	if [ -n "$(1)" ]; then \
		echo "_yarn_tag_install $(1)" >> ${_INTERNAL_BUILD_LOG_}; \
		if ( ${ENJENV_EXE} features has yarn-$(1)--install 2>&1 ) > /dev/null; then \
			${CMD} ${ENJENV_EXE} yarn-$(1)--install; \
		fi; \
	fi
endef

define _yarn_run
	if [ -n "$(1)" -a -n "$(2)" ]; then \
		echo "_yarn_run $(1) $(2)" >> ${_INTERNAL_BUILD_LOG_}; \
		if [ "${HAS_YARN}" == "true" ]; then \
			cd $(1) > /dev/null; \
			${CMD} ${ENJENV_EXE} yarn -- $(2) 2> /dev/null || true; \
			cd - > /dev/null; \
		else \
			echo "# yarn feature not found"; \
			false; \
		fi; \
	fi
endef

define _yarn_run_script
	if [ -n "$(1)" -a -n "$(2)" ]; then \
		echo "_yarn_run_script $(1) $(2)" >> ${_INTERNAL_BUILD_LOG_}; \
		if ( ${ENJENV_EXE} features has yarn-$(1)-$(2) 2>&1 ) > /dev/null; then \
			${CMD} ${ENJENV_EXE} yarn-$(1)-$(2); \
		else \
			echo "# yarn-$(1)-$(2) script not found"; \
			false; \
		fi; \
	fi
endef

define _golang_nancy_installed
	echo "_golang_nancy_installed" >> ${_INTERNAL_BUILD_LOG_}; \
	if [ -n "$(call _has_feature,golang--setup-nancy)" ]; then \
		${CMD} ${ENJENV_EXE} golang setup-nancy || false; \
	fi
endef

define _source_activate_run
	if [ -f "${ENJENV_PATH}/activate" ]; then \
		source "${ENJENV_PATH}/activate" 2>/dev/null \
		&& ${CMD} ${1} ${2} ${3} ${4} ${5} ${6} ${7} ${8} ${9}; \
	else \
		echo "# missing ${ENJENV_PATH}/activate"; \
	fi
endef

define _run_checks
	if [ ! -x "${APP_NAME}" ]; then \
		echo "${APP_NAME} not found or not executable"; \
		false; \
	fi;
endef

ifdef override_run_checks
define _get_run_checks
	$(call override_run_checks)
endef
else
define _get_run_checks
	$(call _run_checks)
endef
endif

define _run
	$(call _get_run_checks) \
	if [ "${DLV_DEBUG}" == "true" -a -n "${DLV_BIN}" ]; then \
		echo "# delving ${APP_NAME} ${RUN_ARGV}"; \
		${CMD} $(call _get_run_vars) \
		${DLV_BIN} --listen=:${DLV_PORT} --headless=true --api-version=2 --accept-multiclient \
			exec -- ./${APP_NAME} ${RUN_ARGV}; \
	else \
		echo "# running ${APP_NAME} ${RUN_ARGV}"; \
		${CMD} $(call _get_run_vars) ./${APP_NAME} ${RUN_ARGV}; \
	fi
endef

define is_defined
$(if $(strip $($1)),true,false)
endef

define _find_pid_tree
`\
	function _all_children { \
		for cid in $$(ps -o pid= --ppid "$${1}"); \
		do \
			echo $$(_all_children "$${cid}"); \
			echo "$${cid}"; \
		done; \
	}; \
	function _up_to_first_make { \
		for ppid in $$(ps -o ppid= -p "$${1}"); \
		do \
			NAME="$$(ps -o command= -p "$${ppid}" | awk '{print $$1 $$2}')"; \
			if echo "$${NAME}" | egrep -q '^(/.*/bash-c|/.*/make.*|make.*)$$'; then \
				echo "$${ppid}"; \
				echo $$(_up_to_first_make "$${ppid}"); \
			fi; \
		done; \
	}; \
	FOUND=""; \
	for pid in $$(_all_children "$(1)") $(1) $$(_up_to_first_make "$(1)") \
	; \
	do \
		if [ -z "$${FOUND}" ]; then \
			FOUND="$${pid}"; \
		else \
			FOUND="$${FOUND} $${pid}"; \
		fi; \
	done; \
	echo "$${FOUND}" \
`
endef

list-make-targets:
	@LC_ALL=C \
		$(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null \
		| awk -v RS= -F: '/(^|\n)# Files(\n|$$)/,/(^|\n)# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
		| sort \
		| egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
		| egrep -v '\-\%$$'
	@$(shell \
if [ -n "${ENJENV_EXE}" -a -x "${ENJENV_EXE}" ]; then \
		_ENJENV_HELP=$$(${ENJENV_EXE} features list); \
		_FOUND=""; \
		if echo "$${_ENJENV_HELP}" | egrep -q '^go-audit$$'; then \
			for tag in ${_LIST_NODE_PATHS}; do \
				echo "echo \"audit-$${tag}\";"; \
			done; \
		fi; \
		for tag in ${_LIST_NODE_PATHS}; do \
			for script in `echo "$${_ENJENV_HELP}" | grep yarn-$${tag}- | awk '{print $$1}'`; do \
				echo "echo \"$${script}\";"; \
				_FOUND="$${_FOUND};$${script}"; \
			done; \
		done; \
		if [ -z "$${_FOUND}" ]; then \
			for tag in ${_LIST_NODE_PATHS}; do \
				echo "echo \"yarn-$${tag}-install\";"; \
			done; \
		fi; \
fi)

#
#: actual make targets
#

help:
	@echo "${APP_NAME} - ${APP_SUMMARY}"
	@echo
	@echo "version: ${VERSION}"
	@echo "release: ${RELEASE}"
	@echo
	@echo "usage: make <target> [targets...]"
	@echo
	@echo "Cleanup Targets:"
	@echo "  clean         remove built binary"
	@echo "  dist-clean    clean and purge local .bin ${ENJENV_DIR}"
	@echo
	@echo "Compile Targets:"
	@echo "  build         build a debug-able version of ${APP_NAME}"
	@echo "  release       build a release version of ${APP_NAME}"
	@echo
	@echo "Runtime Targets:"
	@echo "  dev           set DEBUG mode and run ./${APP_NAME}"
ifneq (${DLV_BIN},)
	@echo "  dlv           same as 'dev' ran with headless dlv (port ${DLV_PORT})"
endif
	@echo "  run           execute ./${APP_NAME}"
	@echo "  stop          interrupt running ${APP_NAME}"
	@echo
	@echo "Aesthetic Targets:"
	@echo "  build-dev-run   make build dev with colourized output"
ifneq (${DLV_BIN},)
	@echo "  build-dlv-run   make build dlv with colourized output"
endif

	@echo
	@echo "#############################################################"
	@echo
	@echo "Profiling Targets:"
	@echo "  profile.cpu   make build dev with cpu profiling"
	@echo "  profile.mem   make build dev with mem profiling"
	@echo
	@echo "  note: use 'make stop' from another terminal to end profiling"

ifneq (${_DEPS_PRESENT},)
	@echo
	@echo "#############################################################"
	@echo
	@echo "Auditing Targets:"
endif
ifneq (${_GO_PRESENT},)
	@echo "  audit		runs enjenv go-audit-report"
endif
ifneq (${_LIST_PACKAGE_JSON},)
	@$(shell \
		for tag in ${_LIST_NODE_PATHS}; do \
			echo "echo \"  audit-$${tag}	runs enjenv $${tag}-audit-report\""; \
		done)
endif

ifneq (${HEROKU_BIN},)
	@echo
	@echo "#############################################################"
	@echo
	@echo "Heroku Targets:"
	@echo "  heroku-tail      wrapper for: heroku logs --tail"
	@echo "  heroku-push      wrapper for: git push ${HEROKU_GIT_REMOTE} ${HEROKU_SRC_BRANCH}:${HEROKU_DST_BRANCH}"
	@echo
	@echo " Configure heroku-push"
	@echo "   HEROKU_GIT_REMOTE=${HEROKU_GIT_REMOTE}"
	@echo "   HEROKU_SRC_REMOTE=${HEROKU_SRC_BRANCH}"
	@echo "   HEROKU_DST_REMOTE=${HEROKU_DST_BRANCH}"
endif

ifneq (${_LIST_PACKAGE_JSON},)
	@echo
	@echo "#############################################################"
	@echo
	@echo "There are nodejs package.json files present in this project"
	@echo "and enjenv has convenient wrappers for a number of extra"
	@echo "targets that Enjin.mk provides. These follow a simple naming"
	@echo "convention: yarn-<dir>-<script-name> where when targetted"
	@echo "with make, will run 'yarn <script-name>' in <dir> for you."
	@echo
	@echo "Yarn Targets:"
	@$(shell \
	if [ -n "${ENJENV_EXE}" -a -x "${ENJENV_EXE}" ]; then \
		_ENJENV_HELP=$$(${ENJENV_EXE} features list); \
		_FOUND=""; \
		for tag in ${_LIST_NODE_PATHS}; do \
			for script in `echo "$${_ENJENV_HELP}" | grep yarn-$${tag}- | awk '{print $$1}'`; do \
				echo "echo \"  $${script}\";"; \
				_FOUND="$${_FOUND};$${script}"; \
			done; \
		done; \
		if [ -z "$${_FOUND}" ]; then \
			for tag in ${_LIST_NODE_PATHS}; do \
				echo "echo \"  yarn-$${tag}-install\";"; \
			done; \
		fi; \
	fi)
	@echo
endif

ifneq (${BE_CONSOLE_KEYS},)
	@echo
	@echo "#############################################################"
	@echo
	@echo "Console Targets:"
	@echo -e "  list-consoles\t\t# (list all console targets)"
	@echo -e "  console\t\t# (run default/first console)"
	@$(if ${BE_CONSOLE_KEYS},$(foreach key,${BE_CONSOLE_KEYS},$(shell \
	_NAME_="$($(key)_CONSOLE_NAME)"; \
	_DESC_="$($(key)_CONSOLE_DESC)"; \
	[ -z "$${_DESC_}" ] && _DESC_="(description not found)"; \
	echo "echo -e \"  con-$${_NAME_}\t# ($${_DESC_})\""; \
)))

endif

	@echo
	@echo "#############################################################"
	@echo
	@echo "Helper Targets:"
	@echo "  help          this screen of output"
	@echo "  tidy          run go mod tidy"
	@echo "  local         go mod -replace for ${GO_ENJIN_PKG}"
	@echo "  unlocal       go mod -dropreplace for ${GO_ENJIN_PKG}"
	@echo "  be-update     go clean and get ${GO_ENJIN_PKG}"
ifeq (${DLV_BIN},)
	@echo "  install-dlv   run go install github.com/go-delve/delve@latest"
endif

	@echo
	@echo "#############################################################"
	@echo
	@echo "This is a Go-Enjin project which depends upon enjenv to"
	@echo "manage the Golang and Nodejs SDKs and build this Enjin."
	@echo
	@echo "Go-Enjin projects use make as their build system and in"
	@echo "particular, they include Enjin.mk (which is where this"
	@echo "help screen lives)."
	@echo
	@echo "Enjin.mk will ensure that any dependencies that are not"
	@echo "already present, will be included just before they're"
	@echo "actually needed. For your information, these are summarized"
	@echo "into two categories, the managed ones Enjin.mk will handle"
	@echo "and the assumptions are up to the developer(s) managing this"
	@echo "project."
	@echo
	@echo "Managed Dependencies:"
	@echo "  enjenv        the Go-Enjin environment utility"
	@echo "  nancy         for auditing go code using Sonatype nancy"
	@echo "  nodejs        enjenv managed nodejs environment"
	@echo "  yarn          preferred nodejs package manager"
	@echo
	@echo "Assumed Dependencies:"
	@echo "  os            linux or darwin"
	@echo "  arch          arm64 or amd64"
	@echo "  make          GNU make is installed"
	@echo "  coreutils     GNU coreutils is installed"
	@echo
	@echo "Enjin.mk Version: ${ENJIN_MK_VERSION}"

_enjenv:
	@if [ -z "${ENJENV_EXE}" -o ! -x "${ENJENV_EXE}" ]; then \
		echo "# critical error: enjenv not found"; \
		false; \
	fi

_golang: _enjenv
	@if [ -z "$(call _has_feature,golang--build)" ]; then \
		if [ "${GOLANG}" != "" ]; then \
			${CMD} ${ENJENV_EXE} golang init --golang "${GOLANG}"; \
		else \
			${CMD} ${ENJENV_EXE} golang init; \
		fi; \
		${CMD} ${ENJENV_EXE} write-scripts; \
		$(call _source_activate_run,${ENJENV_EXE},golang,setup-nancy); \
	elif [ ! -f "${ENJENV_PATH}/activate" ]; then \
		${CMD} ${ENJENV_EXE} write-scripts; \
	else \
		echo "# golang present"; \
	fi

_nodejs: _enjenv
	@if [ -z "$(call _has_feature,yarn)" ]; then \
		if [ "${NODEJS}" != "" ]; then \
			${CMD} ${ENJENV_EXE} nodejs init --nodejs "${NODEJS}"; \
		else \
			${CMD} ${ENJENV_EXE} nodejs init; \
		fi; \
		${CMD} ${ENJENV_EXE} write-scripts; \
	else \
		echo "# nodejs present"; \
	fi

setup-golang: _golang
	@echo "# setup-golang complete"

setup-yarn: _nodejs
	@echo "# setup-yarn complete"

setup: setup-golang setup-yarn

tidy: _golang
	@if [ ${GO_MOD} -le 1017 ]; then \
		echo "# go mod tidy -go=1.16 && go mod tidy -go=1.17"; \
		$(call _source_activate_run,go,mod,tidy,-go=1.16); \
		$(call _source_activate_run,go,mod,tidy,-go=1.17); \
	else \
		echo "# go mod tidy"; \
		$(call _source_activate_run,go,mod,tidy); \
	fi

local: _golang
	@`echo "_make_extra_locals" >> ${_INTERNAL_BUILD_LOG_}`
	@$(call _validate_extra_pkgs)
	@$(if ${GOPKG_KEYS},$(foreach key,${GOPKG_KEYS},$(call _make_go_local,$($(key)_GO_PACKAGE),$($(key)_LOCAL_PATH));))
	@$(call _make_go_local,${GO_ENJIN_PKG},${BE_LOCAL_PATH})

unlocal: _golang
	@`echo "_make_extra_unlocals" >> ${_INTERNAL_BUILD_LOG_}`
	@$(call _validate_extra_pkgs)
	@$(if ${GOPKG_KEYS},$(foreach key,${GOPKG_KEYS},$(call _make_go_unlocal,$($(key)_GO_PACKAGE));))
	@$(call _make_go_unlocal,${GO_ENJIN_PKG})

be-update: export GOPROXY=direct
be-update: PKG_LIST = ${GO_ENJIN_PKG}@latest $(call _make_extra_pkgs)
be-update: _golang
	@$(call _validate_extra_pkgs)
	@echo "# go get ${PKG_LIST}"
	@$(call _source_activate_run,go,get,${_BUILD_TAGS},${PKG_LIST})

clean:
ifdef override_clean
	@$(call override_clean)
else
	@$(call _clean,${CLEAN})
	@$(call _clean,${EXTRA_CLEAN})
endif

dist-clean: clean
ifdef override_dist_clean
	@$(call override_dist_clean)
else
	@if [ -d "./${ENJENV_DIR}" ]; then \
		ENJENV_PATH="${PWD}/${ENJENV_DIR}" \
			${CMD} ${ENJENV_EXE} clean --force; \
	fi
	@$(call _clean,${DIST_CLEAN})
endif

build: export BE_APP_NAME=${APP_NAME}
build: export BE_SUMMARY=${APP_SUMMARY}
build: export BE_ENV_PREFIX=${ENV_PREFIX}
build: export BE_VERSION=${VERSION}
build: export BE_RELEASE=${RELEASE}
build: export ENJENV_GO_LDFLAGS=${_EXTRA_LDFLAGS}
build: export ENJENV_GO_GCFLAGS=${_EXTRA_GCFLAGS}
build: _golang ${EXTRA_BUILD_TARGET_DEPS}
ifdef pre_build
	@$(call pre_build)
endif
ifdef override_build
	@$(call override_build)
else
	@if [ "${PRE_RELEASE_BUILD}" == "true" ]; then \
		echo "# Building pre-release: ${VERSION}, ${RELEASE}"; \
	elif [ "${RELEASE_BUILD}" == "true" ]; then \
		echo "# Building release: ${VERSION}, ${RELEASE}"; \
	else \
		echo "# Building debug: ${VERSION}, ${RELEASE}"; \
	fi
	@echo "## (LDFLAGS): \"${ENJENV_GO_LDFLAGS}\" ; (GCFLAGS): \"${ENJENV_GO_GCFLAGS}\""
	@${CMD} ${ENJENV_EXE} golang build ${_BUILD_ARGS} -- -v ${_BUILD_TAGS} ${_BUILD_ARGV}
	@if [ -x "./${APP_NAME}" ]; then \
		echo "# produced: ${APP_NAME}"; \
		sha256sum ./${APP_NAME}; \
	fi
endif
ifdef post_build
	@$(call post_build)
endif

release: export RELEASE_BUILD=true
release: build
	@$(call _upx_build,"${APP_NAME}")

pre-release: export PRE_RELEASE_BUILD=true
pre-release: build

run: _HAS_O_RUN_=$(call is_defined,override_run)
run: _HAS_O_DLV_=$(call is_defined,override_dlv)
run:
	@echo "# processing run overrides"
ifdef pre_run
	@echo "# calling pre_run"
	@$(call pre_run)
endif
	@if [ "${_HAS_O_RUN_}" == "true" ]; then \
		echo "# override_run present"; \
		if [ -n "${DLV_DEBUG}" ]; then \
			echo "# delving requested"; \
			if [ "${_HAS_O_DLV_}" == "true" ]; then \
				echo "# calling override_dlv"; \
				$(call override_dlv) \
			else \
				echo "error: override_dlv not defined in Makefile"; \
				false; \
			fi; \
		else \
			echo "# normal run requested"; \
			echo "# calling override_run"; \
			$(call override_run) \
		fi; \
	else \
		echo "# override_run not present"; \
		if [ -n "${DLV_DEBUG}" ]; then \
			echo "# delving requested"; \
			if [ "${_HAS_O_DLV_}" == "true" ]; then \
				echo "# calling override_dlv"; \
				$(call override_dlv) \
			else \
				echo "# calling normal run"; \
				$(call _run) \
			fi; \
		else \
			echo "# calling normal run"; \
			$(call _run) \
		fi; \
	fi
ifdef post_run
	@echo "# calling post_run"
	@$(call post_run)
endif

ifneq (${BE_CONSOLE_KEYS},)
.PHONY: list-consoles console con-%

console:
	@_DEFAULT_=$(call _default_console_name); \
	if [ -n "$${_DEFAULT_}" ]; then \
		$(MAKE) DEBUG=${DEBUG} DLV_DEBUG=${DLV_DEBUG} RUN_ARGV="console $${_DEFAULT_}" run; \
	else \
		echo "# default/first console not found"; \
	fi

dlv-console: export DEBUG=true
dlv-console: export DLV_DEBUG=true
dlv-console: console

list-consoles:
	@for name in $(if ${BE_CONSOLE_KEYS},$(foreach key,${BE_CONSOLE_KEYS}, $($(key)_CONSOLE_NAME))); do \
		echo "con-$${name}"; \
	done

con-%: export CON_TAG=$(patsubst con-%,%,$@)
con-%: export RUN_ARGV=console ${CON_TAG}
con-%:
ifdef override_run
	@$(call override_run)
else
	@$(call _run)
endif
endif

dev: DEBUG=true
dev: run

ifneq (${DLV_BIN},)
.PHONY: dlv

dlv: DLV_DEBUG=true
dlv: dev
else
install-dlv: _golang
	@echo "# installing go-delve..."
	@go install github.com/go-delve/delve/cmd/dlv@latest
endif

_audit: _golang
	@$(call _golang_nancy_installed)
	@${CMD} ${ENJENV_EXE} go-audit-report --tags=all

ifneq (${_GO_PRESENT},)
audit: _audit
endif

ifneq (${_LIST_PACKAGE_JSON},)
audit-%: TAG = $(patsubst audit-%,%,$@)
audit-%: _enjenv _nodejs
	@$(call _is_nodejs_tag,${TAG})
	@$(call _yarn_tag_install,${TAG})
	@${CMD} ${ENJENV_EXE} ${TAG}-audit-report

yarn-%: FIRST = $(patsubst yarn-%,%,$@)
yarn-%: SECOND = $(subst -, ,${FIRST})
yarn-%: TAG = $(word 1,${SECOND})
yarn-%: OP = $(patsubst ${TAG}-%,%,${FIRST})
yarn-%: _enjenv _nodejs
	@if [ -z "${TAG}" -o -z "${OP}" -o "${TAG}" == "${OP}" ]; then \
		echo "# invalid yarn target: $@"; \
		false; \
	fi
	@$(call _is_nodejs_tag,${TAG})
	@$(call _yarn_tag_install,${TAG})
	@if [ "${OP}" == "install" -o "${OP}" == "-install" ]; then \
		echo "# yarn-${TAG}--install completed"; \
	else \
		$(call _yarn_run_script,${TAG},${OP}); \
	fi
endif

ifneq (${HEROKU_BIN},)

.PHONY: heroku-logs heroku-push

HEROKU_GIT_REMOTE ?= heroku
HEROKU_SRC_BRANCH ?= trunk
HEROKU_DST_BRANCH ?= main

heroku-push:
	@${CMD} git push ${HEROKU_GIT_REMOTE} ${HEROKU_SRC_BRANCH}:${HEROKU_DST_BRANCH}

heroku-logs:
	@${CMD} heroku logs --tail

endif

# this requires Term::ANSIColor, will error if not present,
# use `make build dev` instead

ifneq (${DLV_BIN},)
.PHONY: build-dlv-run

build-dlv-run: build
	@( $(MAKE) dlv 2>&1 ) | perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'
endif

build-dev-run: build
	@( $(MAKE) dev 2>&1 ) | perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'

build-dev-run-quiet: build
	@( $(MAKE) dev 2>&1 ) \
		| egrep -v '(\s/media/|\s/css/|\.chunk\.)' - \
		| perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'

release-dev-run: release
	@( $(MAKE) dev 2>&1 ) | perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'

pre-release-dev-run: pre-release
	@( $(MAKE) dev 2>&1 ) | perl -p -e 'use Term::ANSIColor qw(colored);while (my $$line = <>) {print STDOUT process_line($$line)."\n";}exit(0);sub process_line {my ($$line) = @_;chomp($$line);if ($$line =~ m!^\[(\d+\-\d+\.\d+)\]\s+([A-Z]+)\s+(.+?)\s*$$!) {my ($$datestamp, $$level, $$message) = ($$1, $$2, $$3);my $$colour = "white";if ($$level eq "ERROR") {$$colour = "bold white on_red";} elsif ($$level eq "INFO") {$$colour = "green";} elsif ($$level eq "DEBUG") {$$colour = "yellow";}my $$out = "[".colored($$datestamp, "blue")."]";$$out .= " ".colored($$level, $$colour);if ($$level eq "DEBUG") {$$out .= "\t";if ($$message =~ m!^(.+?)\:(\d+)\s+\[(.+?)\]\s+(.+?)\s*$$!) {my ($$file, $$ln, $$tag, $$info) = ($$1, $$2, $$3, $$4);$$out .= colored($$file, "bright_blue");$$out .= ":".colored($$ln, "blue");$$out .= " [".colored($$tag, "bright_blue")."]";$$out .= " ".colored($$info, "bold cyan");} else {$$out .= $$message;}} elsif ($$level eq "ERROR") {$$out .= "\t".colored($$message, $$colour);} elsif ($$level eq "INFO") {$$out .= "\t".colored($$message, $$colour);} else {$$out .= "\t".$$message;}return $$out;}return $$line;}'

stop:
	@RUNNING_PIDS=$$(\
		COLUMNS=1024 ps -x -a -o pid=,command= \
			| egrep -v '(grep|tail)' \
			| egrep "^\\s*[0-9]*\\s*\./${APP_NAME}\$$" \
			| awk '{print $$1}' \
		); \
		if [ -z "$${RUNNING_PIDS}" ]; then \
			echo "# no ${APP_NAME} processes found, nothing to stop"; \
		else \
			for RP in $${RUNNING_PIDS}; do \
				RP_WD=$$( lsof -a -p $${RP} -d cwd -F n | tail -1 | cut -c2- ); \
				LINE=$$(\
					COLUMNS=1024 ps -x -a -o pid=,command= \
						| egrep -v '(grep|tail)' \
						| egrep "^\\s*$${RP}\\s*\./${APP_NAME}\$$" \
				); \
				RP_TREE=$(call _find_pid_tree,$${RP}); \
				echo "#######################################################"; \
				echo "# workdir: $${RP_WD}"; \
				echo "# process: $${LINE}"; \
				if [ -z "${STOP_PROFILING}" ]; then \
				echo "# pidtree:"; \
					for pid in $${RP_TREE}; do \
							INFO=$$(\
								COLUMNS=25 \
								ps -o command= -p $${pid} \
									| perl -pe 's!(^\s*|\s*$$)!!g;$$v=$$_;$$_="";print $$v;print "..." unless (length($$v) <= 24);' \
							); \
							echo "#          $${pid} - $${INFO}"; \
					done; \
				fi; \
				echo "#"; \
				if [ -z "${STOP_ALL}" -a -n "$${RP_WD}" -a "$${RP_WD}" != "$${PWD}" ]; then \
					echo "# skipped (not this path: $${PWD})"; \
					echo ""; \
					continue; \
				fi; \
				rv=0; \
				if [ -z "${FORCE_STOP}" ]; then \
					read -n 1 -p "# terminate? [Yn] " ANSWER; \
					rv=$$?; \
				else \
					echo "# terminate? [Yn] y"; \
				fi; \
				if [ $${rv} -eq 0 ]; then \
					[ -n "$${ANSWER}" ] && echo ""; \
					if [ -z "$${ANSWER}" -o "$${ANSWER}" == "y" -o "$${ANSWER}" == "Y" ]; then \
						if [ -z "${STOP_PROFILING}" ]; then \
							for pid in $${RP_TREE}; do \
								${CMD} kill -KILL $${pid} 2> /dev/null; \
							done; \
							echo "# terminated: $${RP_TREE}"; \
						else \
							${CMD} kill -TERM $${RP} 2> /dev/null; \
							echo "# terminated: $${RP}"; \
						fi; \
					fi; \
				else \
					echo "# stop target aborted"; \
				fi; \
				echo ""; \
			done; \
		fi

stop-all: STOP_ALL=true
stop-all: stop

force-stop: FORCE_STOP=true
force-stop: stop

force-stop-all: STOP_ALL=true
force-stop-all: FORCE_STOP=true
force-stop-all: stop

stop-profiling: STOP_PROFILING=true
stop-profiling: stop

profile.mem: export BE_PROFILE_MODE=mem
profile.mem: export BE_PROFILE_PATH=.
profile.mem: build dev
	@if [ -f mem.pprof ]; then \
		echo "# <Enter> to load mem.pprof, <CTRL+c> to abort"; \
		read JUNK; \
		echo "# pprof service starting (:8080)"; \
		bash -c 'set -m; ( go tool pprof -http=:8080 mem.pprof ) 2> /dev/null'; \
		echo ""; \
		echo "# pprof service shutdown"; \
	else \
		echo "# missing mem.pprof"; \
	fi

profile.cpu: export BE_PROFILE_MODE=cpu
profile.cpu: export BE_PROFILE_PATH=${PROFILE_PATH}
profile.cpu: build dev
	@if [ -f cpu.pprof ]; then \
		echo "# <Enter> to load cpu.pprof, <CTRL+c> to abort"; \
		read JUNK; \
		echo "# pprof service starting (:8080)"; \
		bash -c 'set -m; ( go tool pprof -http=:8080 cpu.pprof ) 2> /dev/null'; \
		echo ""; \
		echo "# pprof service shutdown"; \
	else \
		echo "# missing cpu.pprof"; \
	fi