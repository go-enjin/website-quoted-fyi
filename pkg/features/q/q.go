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

package q

import (
	"github.com/go-corelibs/context"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/indexing/search"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

const (
	Tag feature.Tag = "quote-page-format"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

func init() {
	search.RegisterSearchPageType("quote")
}

type Feature interface {
	feature.Feature
	feature.PageFormat
	feature.FuncMapProvider
	feature.QueryIndexSourceFeature
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature

	eql feature.QueryIndexFeature
}

func New() MakeFeature {
	f := new(CFeature)
	f.Init(f)
	f.PackageTag = Tag
	f.FeatureTag = Tag
	f.CFeature.Construct(f)
	return f
}

func (f *CFeature) Init(this interface{}) {
	f.CFeature.Init(this)
}

func (f *CFeature) Make() Feature {
	return f
}

func (f *CFeature) Setup(enjin feature.Internals) {
	f.CFeature.Setup(enjin)
}

func (f *CFeature) Name() (name string) {
	name = "q"
	return
}

func (f *CFeature) Extensions() (extensions []string) {
	extensions = append(extensions, "q", "q.tmpl")
	return
}

func (f *CFeature) MakeFuncMap(ctx context.Context) (fm feature.FuncMap) {
	return feature.FuncMap{
		"flattenContent": quote.FlattenContent,
	}
}
