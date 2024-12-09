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

package data

import (
	"encoding/json"
	"fmt"

	"vitess.io/vitess/go/test/endtoend/cluster"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

type hashVindex struct {
	vindexes.Hash
	Type string `json:"type"`
}

func (hv hashVindex) String() string {
	return "xxhash"
}

func defaultVschema(defaultKeyspaceName string) *vindexes.VSchema {
	return &vindexes.VSchema{
		Keyspaces: map[string]*vindexes.KeyspaceSchema{
			defaultKeyspaceName: {
				Keyspace: &vindexes.Keyspace{},
				Tables:   map[string]*vindexes.Table{},
				Vindexes: map[string]vindexes.Vindex{
					"xxhash": &hashVindex{Type: "xxhash"},
				},
				Views: map[string]sqlparser.SelectStatement{},
			},
		},
	}
}

func GetKeyspaces(vschemaFile, vtexplainVschemaFile, keyspaceName string, sharded bool) (keyspaces []*cluster.Keyspace, vschema *vindexes.VSchema, err error) {
	ksRaw := RawKeyspaceVindex{
		Keyspaces: map[string]interface{}{},
	}

	switch {
	case vschemaFile != "":
		ksRaw, vschema, err = ReadVschema(vschemaFile, false)
		if err != nil {
			return nil, nil, fmt.Errorf("reading vschema: %w", err)
		}
	case vtexplainVschemaFile != "":
		ksRaw, vschema, err = ReadVschema(vtexplainVschemaFile, true)
		if err != nil {
			return nil, nil, fmt.Errorf("reading vtexplain vschema: %w", err)
		}
	default:
		// auto-vschema
		vschema = defaultVschema(keyspaceName)
		vschema.Keyspaces[keyspaceName].Keyspace.Sharded = sharded
		ksSchema, err := json.Marshal(vschema.Keyspaces[keyspaceName])
		if err != nil {
			return nil, nil, fmt.Errorf("marshalling keyspace schema: %w", err)
		}
		ksRaw.Keyspaces[keyspaceName] = ksSchema
	}

	for key, value := range ksRaw.Keyspaces {
		var ksSchema string
		valueRaw, ok := value.([]uint8)
		if !ok {
			valueRaw, err = json.Marshal(value)
			if err != nil {
				return nil, nil, fmt.Errorf("marshalling keyspace schema: %w", err)
			}
		}
		ksSchema = string(valueRaw)
		keyspaces = append(keyspaces, &cluster.Keyspace{
			Name:    key,
			VSchema: ksSchema,
		})
	}
	return keyspaces, vschema, nil
}
