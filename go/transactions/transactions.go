/*
Copyright 2024 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transactions

import (
	"github.com/vitessio/vt/go/data"
	"vitess.io/vitess/go/vt/sqlparser"
)

type (
	Config struct {
		FileName string
		Loader   data.Loader
	}

	Connection struct {
		// The connection ID
		ID int

		buf []data.Query

		Autocommit bool
	}

	TxSignature struct {
	}
)

func Run(cfg Config) {
	// Figure out if autocommit is enabled
	// If we see:
	// 1. BEGIN we can assume autocommit is disabled
	// 2. COMMIT and no BEGIN we can assume autocommit is enabled
	// 3. ROLLBACK and no BEGIN we can assume autocommit is enabled
	// 4. SET autocommit = 1/0
	count := 1000
	defaultAutocommit := true
	loader := cfg.Loader.Load(cfg.FileName)
	for {
		count--
		if count == 0 {
			// enough already. we'll assume autocommit is enabled because that is the default
			break
		}
		query, kontinue := loader.Next()
		if !kontinue {
			break
		}

		switch query.Type {
		case data.Skip, data.Error, data.VExplain, data.Unknown:
			panic("unexpected query type")
		case data.Comment, data.CommentWithCommand, data.EmptyLine, data.WaitForAuthoritative, data.SkipIfBelowVersion:
			// no-op for keys
		case data.QueryT:
			stmt, err := sqlparser.NewTestParser().Parse(query.Query)
			if err != nil {
				continue
			}
			switch stmt.(type) {
			case *sqlparser.Begin:
				defaultAutocommit = false
				break
			case *sqlparser.Commit:
				break
			}
		}
	}
	err := loader.Close()
	if err != nil {
		panic(err.Error())
	}

	connections := map[int]*Connection{}

	loader = cfg.Loader.Load(cfg.FileName)
	for {

	}
}
