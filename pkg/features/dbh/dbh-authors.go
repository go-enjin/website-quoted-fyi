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

package dbh

func (f *CFeature) GetAuthorNames(nameOrKey string) (fullName, lastName, flat string, ok bool) {
	if _, results, ee := f.eql.PerformLookup(
		`LOOKUP author.Flat, author.FullName, author.LastName WITHIN (author.Flat == %q) OR (author.FullName == %q)`,
		nameOrKey, nameOrKey,
	); ee == nil && len(results) > 0 {
		if values := results[0].SelectStringValues("FullName", "LastName", "Flat"); len(values) == 3 {
			fullName, lastName, flat = values[0], values[1], values[2]
		}
		ok = fullName != "" && lastName != "" && flat != ""
	}
	return
}

func (f *CFeature) GetAuthorKeyFrom(nameOrKey string) (flat string, ok bool) {
	if _, results, ee := f.eql.PerformLookup(
		`LOOKUP author.Flat WITHIN (author.Flat == %q) OR (author.FullName == %q)`,
		nameOrKey, nameOrKey,
	); ee == nil && len(results) > 0 {
		flat = results.FirstStringValue("Flat")
		ok = flat != ""
	}
	return
}

func (f *CFeature) GetAuthorNameFrom(nameOrKey string) (fullName string, ok bool) {
	if _, results, ee := f.eql.PerformLookup(
		`LOOKUP author.FullName WITHIN (author.Flat == %q) OR (author.FullName == %q)`,
		nameOrKey, nameOrKey,
	); ee == nil && len(results) > 0 {
		fullName = results.FirstStringValue("FullName")
		ok = fullName != ""
	}
	return
}
