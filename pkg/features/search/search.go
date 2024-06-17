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

package search

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/maruel/natural"
	"github.com/urfave/cli/v2"

	"github.com/go-corelibs/context"
	"github.com/go-corelibs/maps"
	"github.com/go-corelibs/rxp"
	clStrings "github.com/go-corelibs/strings"
	"github.com/go-corelibs/x-text/language"
	"github.com/go-enjin/be/features/pages/search"
	"github.com/go-enjin/be/pkg/feature"
	"github.com/go-enjin/be/pkg/log"
	"github.com/go-enjin/website-quoted-fyi/pkg/features/dbh"
	"github.com/go-enjin/website-quoted-fyi/pkg/quote"
)

var (
	_ Feature     = (*CFeature)(nil)
	_ MakeFeature = (*CFeature)(nil)
)

const Tag feature.Tag = "search-quoted"

type Feature interface {
	feature.Feature
	search.ResultsPostProcessor
	feature.SearchEnjinFeature
}

type MakeFeature interface {
	Make() Feature
}

type CFeature struct {
	feature.CFeature

	dbh dbh.Feature
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

func (f *CFeature) Startup(ctx *cli.Context) (err error) {
	if err = f.CFeature.Startup(ctx); err != nil {
		return
	}

	if found := f.Enjin.GetQueryIndexFeatures(); len(found) > 0 {
		f.eql = found[0]
	} else {
		err = fmt.Errorf("%v feature requires at least one feature.QueryIndexFeature", f.Tag())
		return
	}

	if f.dbh = feature.FirstTyped[dbh.Feature](f.Enjin.Features().List()); f.dbh == nil {
		err = fmt.Errorf("%v features requires dbh.Feature", f.Tag())
		return
	}

	return
}

func (f *CFeature) PostStartup(ctx *cli.Context) (err error) {

	if tfs := feature.FilterTyped[dbh.Feature](f.Enjin.Features().List()); len(tfs) > 0 {
		f.dbh = tfs[0]
	} else {
		err = fmt.Errorf("a dbh.Feature is required")
		return
	}

	return
}

func (f *CFeature) PrepareSearch(tag language.Tag, input string) (query string) {
	keywords := rxp.Pattern{rxp.IsKeyword("c")}.FindAllString(input, -1)
	for idx, keyword := range keywords {
		keyword = strings.ToLower(keyword)
		if idx > 0 {
			query += " "
		}
		query += keyword
	}
	return
}

func (f *CFeature) PerformSearch(tag language.Tag, input string, size, pg int) (bsr *bleve.SearchResult, err error) {
	if size <= 0 && pg < 0 {
		log.ErrorF("size <= 0 && pg < 0 - how did this happen?")
		return nil, fmt.Errorf("invalid pagination settings")
	}

	// parse the input string into a list of keywords
	keywords := rxp.Pattern{rxp.IsKeyword("c")}.FindAllString(input, -1)
	var order []string
	unique := make(map[string]struct{})
	// categorize the parsed keywords into must/should/not mappings to their keywords indexes
	// not sure the idx weights are necessary, scoring is done within query
	mustWords, shouldWords, notWords := make(map[string]int), make(map[string]int), make(map[string]int)
	for idx, keyword := range keywords {
		if keyword = strings.ToLower(keyword); keyword != "" {
			kw := keyword
			switch keyword[0] {
			case '+':
				kw = keyword[1:]
				if _, present := mustWords[kw]; !present {
					mustWords[kw] = idx
				}
			case '-':
				kw = keyword[1:]
				if _, present := notWords[kw]; !present {
					notWords[kw] = idx
				}
			default:
				if _, present := shouldWords[kw]; !present {
					shouldWords[kw] = idx
				}
			}
			if _, present := unique[kw]; !present {
				unique[kw] = struct{}{}
				order = append(order, kw)
			}
			keywords[idx] = kw
		}
	}

	if len(mustWords) == 0 && len(shouldWords) == 0 {
		if len(notWords) > 0 {
			return nil, fmt.Errorf("must have at least one matching search term")
		}
		return nil, fmt.Errorf("empty search query")
	}

	var mustWids, shouldWids, notWids []string
	wordList, lookupWid, lookupFlat, _ /* lookupWord */ := f.dbh.GetWordInfoFrom(order...)
	for idx, word := range wordList {
		kw := order[idx]
		if flat, ok := lookupFlat[word]; ok {
			if widint, ok := lookupWid[flat]; ok {
				wid := fmt.Sprintf("%d", widint)
				if _, present := mustWords[kw]; present {
					mustWids = append(mustWids, wid)
				} else if _, present = notWords[kw]; present {
					notWids = append(notWids, wid)
				} else {
					shouldWids = append(shouldWids, wid)
				}
			}
		}
	}

	var total int
	var query string

	// "+80 +1977 -8 -26"
	// 80, 1977 are +terms
	// 8, 26    are -terms
	query = `SELECT`
	query += ` found.id, p.url, p.shasum, a.full_name AS author, q.hash,`

	if len(mustWords) > 0 {
		// musts, maybe shoulds and/or nots

		// select the total possible results
		countSql := `SELECT`
		countSql += `   COUNT(DISTINCT(p.id)) AS total`
		countSql += `   FROM qf_eql_page AS p`
		for idx, wid := range mustWids {
			if idx == 0 {
				countSql += `   WHERE p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + wid + `)`
			} else {
				countSql += `   AND   p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + wid + `)`
			}
		}
		if count := len(notWids); count > 1 {
			countSql += `   AND p.id NOT IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id IN (`
			countSql += strings.Join(notWids, ",")
			countSql += `   ))`
		} else if count == 1 {
			countSql += `   AND p.id NOT IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + notWids[0] + `)`
		}
		countSql += `;`

		if _, rslts, ee := f.eql.EQL().SqlQuery(countSql); ee == nil {
			total = rslts[0].Int("total")
		} else {
			log.ErrorF("error selecting count of should keyword query: %q - %v", countSql, ee)
			err = ee
			return
		}

		// calculate the score
		if mc, sc := len(mustWids), len(shouldWids); mc == 1 && sc == 0 {

			// only one must-word, can simplify query
			query += ` (SELECT IFNULL(SUM(qw.tally), 0) FROM qf_eql_quote_word AS qw WHERE qw.page_id = found.id AND qw.word_id = ` + mustWids[0] + `) AS score`

		} else if mc >= 1 {
			// one or more must-words and/or should-words

			numWids := mc + sc
			for idx, wid := range append(mustWids, shouldWids...) {
				if idx > 0 {
					query += ` + `
				}
				weight := numWids - idx
				query += `(SELECT IFNULL(SUM(qw.tally + ` + strconv.Itoa(weight) + `), 0)`
				query += ` FROM qf_eql_quote_word AS qw WHERE qw.page_id = found.id AND qw.word_id = ` + wid + `)`
			}
			query += ` AS score`

		} else {
			// not sure how this can happen but let's make sure a non-null zero score exists in all queries
			query += ` 0 AS score`
		}

		query += ` FROM`
		query += ` (SELECT`
		query += `   DISTINCT(p.id) AS id`
		query += `   FROM qf_eql_page AS p`

		for idx, wid := range mustWids {
			if idx == 0 {
				query += `   WHERE p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + wid + `)`
			} else {
				query += `   AND   p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + wid + `)`
			}
		}

	} else {
		// only shoulds and maybe nots

		// select the total possible results
		countSql := `SELECT`
		countSql += `   COUNT(DISTINCT(p.id)) AS total`
		countSql += `   FROM qf_eql_page AS p`
		countSql += `   WHERE p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id IN (` + strings.Join(shouldWids, ",") + `))`
		if count := len(notWids); count > 1 {
			countSql += `   AND p.id NOT IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id IN (`
			countSql += strings.Join(notWids, ",")
			countSql += `   ))`
		} else if count == 1 {
			countSql += `   AND p.id NOT IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + notWids[0] + `)`
		}

		if _, rslts, ee := f.eql.EQL().SqlQuery(countSql); ee == nil {
			total = rslts[0].Int("total")
		} else {
			log.ErrorF("error selecting count of should keyword query: %q - %v", countSql, ee)
			err = ee
			return
		}

		query = `SELECT`
		query += ` found.id, p.url, p.shasum, a.full_name AS author, q.hash,`

		// calculate score
		sc := len(shouldWids)
		for idx, wid := range shouldWids {
			if idx > 0 {
				query += ` + `
			}
			weight := sc - idx
			query += `(SELECT IFNULL(SUM(qw.tally + ` + strconv.Itoa(weight) + `), 0)`
			query += ` FROM qf_eql_quote_word AS qw WHERE qw.page_id = found.id AND qw.word_id = ` + wid + `)`
		}
		query += ` AS score`

		// from, where, etc
		query += ` FROM`
		query += ` (SELECT`
		query += `   DISTINCT(p.id) AS id`
		query += `   FROM qf_eql_page AS p`
		if len(shouldWids) == 1 {
			query += `   WHERE p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + shouldWids[0] + `)`
		} else {
			query += `   WHERE p.id IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id IN (` + strings.Join(shouldWids, ",") + `))`
		}
	}

	if count := len(notWids); count > 1 {
		query += `   AND p.id NOT IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id IN (`
		query += strings.Join(notWids, ",")
		query += `   ))`
	} else if count == 1 {
		query += `   AND p.id NOT IN (SELECT page_id FROM qf_eql_quote_word WHERE word_id = ` + notWids[0] + `)`
	}
	query += ` ) AS found`
	query += ` INNER JOIN qf_eql_page         AS p  ON p.id        = found.id`
	query += ` INNER JOIN qf_eql_quote        AS q  ON q.page_id   = found.id`
	query += ` INNER JOIN qf_eql_quote_word   AS qw ON qw.page_id  = found.id`
	query += ` INNER JOIN qf_eql_quote_author AS qa ON qa.page_id  = found.id`
	query += ` INNER JOIN qf_eql_author       AS a  ON a.id        = qa.author_id`
	query += ` GROUP BY found.id`
	query += ` ORDER BY score DESC, found.id`
	query += ` LIMIT ` + strconv.Itoa(size)
	query += ` OFFSET ` + strconv.Itoa(size*pg)
	query += `;`

	var results context.Contexts
	if _, results, err = f.eql.EQL().SqlQuery(query); err != nil {
		log.ErrorF("error performing sql query: %q - %v", query, err)
		return
	}

	// prepare return values

	var hits []*bleveSearch.DocumentMatch
	var maxScore float64
	for idx, result := range results {
		url := result.String("url")
		score := result.Float64("score")
		if maxScore < score {
			maxScore = score
		}
		hit := &bleveSearch.DocumentMatch{
			Index:     url,
			ID:        url,
			Score:     score,
			HitNumber: uint64(idx + 1),
			Fields: map[string]interface{}{
				"url":    url,
				"score":  score,
				"hash":   result.String("hash"),
				"shasum": result.String("shasum"),
				"author": result.String("author"),
			},
		}
		hits = append(hits, hit)
	}

	bsr = &bleve.SearchResult{
		Status: &bleve.SearchStatus{
			Total:      total,
			Failed:     0,
			Successful: total,
		},
		Hits:     hits,
		Total:    uint64(total),
		Request:  nil,
		MaxScore: maxScore,
	}
	return
}

func (f *CFeature) SearchResultsPostProcess(r *http.Request, p feature.Page) {
	var query string
	if query = p.Context().String("SiteSearchQuery", ""); query == "" {
		p.SetTitle("Quoted.FYI: Search")
	} else {
		p.SetTitle("Quoted.FYI: Searching")
	}
	p.Context().SetSpecific("Title", p.Title())

	if results, ok := p.Context().Get("SiteSearchResults").(*bleve.SearchResult); ok {

		authorLookup := make(map[string]*struct {
			s float64
			q []*quote.Quote
		})

		for _, hit := range results.Hits {
			url := hit.Fields["url"].(string)
			hash := hit.Fields["hash"].(string)
			score := hit.Fields["score"].(float64)
			//shasum := 	hit.Fields["shasum"].(string)
			author := hit.Fields["author"].(string)
			if _, present := authorLookup[author]; !present {
				authorLookup[author] = &struct {
					s float64
					q []*quote.Quote
				}{s: score, q: []*quote.Quote{}}
			}
			authorLookup[author].s += score
			authorLookup[author].q = append(authorLookup[author].q, &quote.Quote{
				Url:  url,
				Hash: hash,
			})
		}

		authorNames := maps.Keys(authorLookup)
		quotedGroups := make([]*quote.AuthorsGroup, 0)

		for _, authorName := range authorNames {
			authorKey := quote.FlattenContent(authorName)
			quotedGroups = append(quotedGroups, &quote.AuthorsGroup{
				Key: fmt.Sprintf("%v", authorLookup[authorName].s),
				Authors: []*quote.Author{{
					Key:    authorKey,
					Name:   authorName,
					Last:   clStrings.LastName(authorName),
					Quotes: authorLookup[authorName].q,
				}},
			})
		}

		sort.Slice(quotedGroups, func(i, j int) (less bool) {
			a, b := quotedGroups[i].Authors[0], quotedGroups[j].Authors[0]
			as, bs := authorLookup[a.Name].s, authorLookup[b.Name].s
			if as == bs {
				if a.Last == b.Last {
					// sort by full name
					return natural.Less(a.Name, b.Name)
				}
				// sort by last names
				return natural.Less(a.Last, b.Last)
			}
			return as > bs // descending score order (greater scores are more relevant)
		})

		p.Context().SetSpecific("QuotedGroups", quotedGroups)

	}
}

func (f *CFeature) AddToSearchIndex(stub *feature.PageStub, p feature.Page) (err error) {
	log.WarnF("%v feature does not support adding pages to the words index", f.Tag())
	return
}

func (f *CFeature) RemoveFromSearchIndex(stub *feature.PageStub, p feature.Page) {
	log.WarnF("%v feature does not support removing pages from the words index", f.Tag())
	return
}
