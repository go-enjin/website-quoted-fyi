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

import (
	"fmt"
)

func Join[V interface{}](slice []V, delim string) (joined string) {
	// Join should be moved to go-corelibs/strings or go-corelibs/slices?
	if size := len(slice); size == 1 {
		joined = fmt.Sprintf("%v", slice[0])
		return
	} else if size < 1 {
		return
	}
	for idx, item := range slice {
		if idx > 0 {
			joined += delim
		}
		joined += fmt.Sprintf("%v", item)
	}
	return
}
