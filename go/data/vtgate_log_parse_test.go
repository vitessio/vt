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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVtGateQueryLog(t *testing.T) {
	gotQueries, err := VtGateLogLoader{}.Load("./testdata/vtgate.query.log")
	require.NoError(t, err)
	require.Len(t, gotQueries, 25)

	require.Equal(t, "vexplain trace insert into pincode_areas(pincode, area_name) values (110001, 'Connaught Place'), (110002, 'Lodhi Road'), (110003, 'Civil Lines'), (110004, 'Kashmere Gate'), (110005, 'Chandni Chowk'), (110006, 'Barakhamba Road'), (110007, 'Kamla Nagar'), (110008, 'Karol Bagh'), (110009, 'Paharganj'), (110010, 'Patel Nagar'), (110011, 'South Extension'), (110012, 'Lajpat Nagar'), (110013, 'Sarojini Nagar'), (110014, 'Malviya Nagar'), (110015, 'Saket')", gotQueries[4].Query)
}
