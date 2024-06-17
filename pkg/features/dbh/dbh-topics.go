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

func (f *CFeature) GetTopicNames(nameOrKey string) (topic, key string, ok bool) {
	if _, results, ee := f.eql.PerformLookup(
		`LOOKUP topic.Flat, topic.Topic WITHIN (topic.Flat == %q) OR (topic.Topic == %q)`,
		nameOrKey, nameOrKey,
	); ee == nil && len(results) > 0 {
		if values := results[0].SelectStringValues("Topic", "Flat"); len(values) == 2 {
			topic, key = values[0], values[1]
		}
		ok = topic != "" && key != ""
	}
	return
}

func (f *CFeature) GetTopicKeyFrom(nameOrKey string) (flat string, ok bool) {
	if _, results, ee := f.eql.PerformLookup(
		`LOOKUP topic.Flat WITHIN (topic.Flat == %q) OR (topic.Topic == %q)`,
		nameOrKey, nameOrKey,
	); ee == nil && len(results) > 0 {
		flat = results.FirstStringValue("flat")
		ok = flat != ""
	}
	return
}

func (f *CFeature) GetTopicNameFrom(nameOrKey string) (topic string, ok bool) {
	if _, results, ee := f.eql.PerformLookup(
		`LOOKUP topic.Topic WITHIN (topic.Flat == %q) OR (topic.Topic == %q)`,
		nameOrKey, nameOrKey,
	); ee == nil && len(results) > 0 {
		topic = results.FirstStringValue("topic")
		ok = topic != ""
	}
	return
}
