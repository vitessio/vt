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

package vitess_tester

import (
	"fmt"
	"os"
	"os/exec"
)

var environmentVars = []string{
	"VTDATAROOT",
	"VTROOT",
}
var neededBinaries = []string{
	"vtgate",
	"vttablet",
	"vtctldclient",
	"mysqlctl",
	"mysqld",
	"etcd",
}

// CheckEnvironment checks if the required environment variables are set
func CheckEnvironment() error {
	for _, envVar := range environmentVars {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("environment variable %s is not set\nTry sourcing the dev.env file in the vitess directory", envVar)
		}
	}

	for _, binary := range neededBinaries {
		_, err := exec.LookPath(binary)
		if err != nil {
			return fmt.Errorf("binary %s not found in PATH", binary)
		}
	}

	return nil
}
