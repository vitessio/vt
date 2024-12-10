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
	"errors"
	"fmt"
	"os"

	vschemapb "vitess.io/vitess/go/vt/proto/vschema"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vtgate/vindexes"
)

type RawKeyspaceVindex struct {
	Keyspaces map[string]interface{} `json:"keyspaces"`
}

func ReadVschema(file string, vtexplain bool) (RawKeyspaceVindex, *vindexes.VSchema, error) {
	rawVschema, srvVschema, err := getSrvVschema(file, vtexplain)
	if err != nil {
		return RawKeyspaceVindex{}, nil, err
	}
	return loadVschema(srvVschema, rawVschema)
}

func getSrvVschema(file string, wrap bool) ([]byte, *vschemapb.SrvVSchema, error) {
	vschemaStr, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}

	if wrap {
		vschemaStr = []byte(fmt.Sprintf(`{"keyspaces": %s}`, vschemaStr))
	}

	var srvVSchema vschemapb.SrvVSchema
	err = json.Unmarshal(vschemaStr, &srvVSchema)
	if err != nil {
		return nil, nil, err
	}

	if len(srvVSchema.Keyspaces) == 0 {
		return nil, nil, errors.New("no keyspace found")
	}

	return vschemaStr, &srvVSchema, nil
}

func loadVschema(srvVschema *vschemapb.SrvVSchema, rawVschema []byte) (RawKeyspaceVindex, *vindexes.VSchema, error) {
	vschema := vindexes.BuildVSchema(srvVschema, sqlparser.NewTestParser())
	if len(vschema.Keyspaces) == 0 {
		return RawKeyspaceVindex{}, nil, errors.New("no keyspace defined in vschema")
	}

	var rkv RawKeyspaceVindex
	err := json.Unmarshal(rawVschema, &rkv)
	if err != nil {
		return RawKeyspaceVindex{}, nil, err
	}
	return rkv, vschema, nil
}
