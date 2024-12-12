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

package summarize

import (
	"github.com/vitessio/vt/go/planalyze"
)

func summarizePlanAnalyze(s *Summary, data planalyze.Output) (err error) {
	s.PlanAnalysis = PlanAnalysis{
		PassThrough:  len(data.PassThrough),
		SimpleRouted: len(data.SimpleRouted),
		Complex:      len(data.Complex),
		Unplannable:  len(data.Unplannable),
	}
	s.PlanAnalysis.Total = s.PlanAnalysis.PassThrough + s.PlanAnalysis.SimpleRouted + s.PlanAnalysis.Complex + s.PlanAnalysis.Unplannable

	s.addPlanResult(data.SimpleRouted)
	s.addPlanResult(data.Complex)

	s.PlanAnalysis.SimpleRoutedQ = append(s.PlanAnalysis.SimpleRoutedQ, data.SimpleRouted...)
	s.PlanAnalysis.ComplexQ = append(s.PlanAnalysis.ComplexQ, data.Complex...)
	return nil
}
