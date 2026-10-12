package matching

import (
	"encoding/json"
	"fmt"
	"io"
)

const AnalysisSchemaVersion = "lumi.event.match_analysis.v1"

// Fixture is a set of labeled query/candidate cases used to measure the
// matcher. Expected labels are intentionally separate from source records.
type Fixture struct {
	Cases []EvaluationCase `json:"cases"`
}

type EvaluationCase struct {
	ID         string   `json:"id"`
	Query      Event    `json:"query"`
	Candidates []Event  `json:"candidates"`
	Expected   Expected `json:"expected"`
}

type Expected struct {
	Status      Status `json:"status"`
	CandidateID string `json:"candidate_id,omitempty"`
}

type CaseResult struct {
	ID          string   `json:"id"`
	Expected    Expected `json:"expected"`
	Decision    Decision `json:"decision"`
	Correct     bool     `json:"correct"`
	Top1Correct bool     `json:"top1_correct"`
}

type Summary struct {
	TotalCases          int                       `json:"total_cases"`
	ExpectedMatched     int                       `json:"expected_matched"`
	PredictedMatched    int                       `json:"predicted_matched"`
	CorrectAutoMatches  int                       `json:"correct_auto_matches"`
	FalseMerges         int                       `json:"false_merges"`
	MissedMatches       int                       `json:"missed_matches"`
	AmbiguousCases      int                       `json:"ambiguous_cases"`
	TieAlerts           int                       `json:"tie_alerts"`
	StatusCounts        map[string]int            `json:"status_counts"`
	ExpectedStatusCount map[string]int            `json:"expected_status_counts"`
	StatusConfusion     map[string]map[string]int `json:"status_confusion"`
	AlertReasons        map[string]int            `json:"alert_reasons"`
	AutoMatchPrecision  float64                   `json:"auto_match_precision"`
	AutoMatchRecall     float64                   `json:"auto_match_recall"`
	AutoMatchF1         float64                   `json:"auto_match_f1"`
	Top1Accuracy        float64                   `json:"top1_accuracy"`
	AmbiguousRate       float64                   `json:"ambiguous_rate"`
}

type Report struct {
	SchemaVersion string       `json:"schema_version"`
	Config        Config       `json:"config"`
	Summary       Summary      `json:"summary"`
	Cases         []CaseResult `json:"cases"`
}

// LoadFixture reads the stable JSON fixture format used by the CLI and tests.
func LoadFixture(reader io.Reader) (Fixture, error) {
	var fixture Fixture
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&fixture); err != nil {
		return Fixture{}, fmt.Errorf("decode fixture: %w", err)
	}
	if len(fixture.Cases) == 0 {
		return Fixture{}, fmt.Errorf("fixture has no cases")
	}
	return fixture, nil
}

// Evaluate runs every labeled case and calculates both coverage and accuracy
// of automatic merges. An ambiguous result is not counted as a merge.
func Evaluate(fixture Fixture, cfg Config) Report {
	summary := Summary{
		TotalCases:          len(fixture.Cases),
		StatusCounts:        map[string]int{},
		ExpectedStatusCount: map[string]int{},
		StatusConfusion:     map[string]map[string]int{},
		AlertReasons:        map[string]int{},
	}
	results := make([]CaseResult, 0, len(fixture.Cases))
	top1Total := 0
	top1Correct := 0

	for _, testCase := range fixture.Cases {
		decision := Match(testCase.Query, testCase.Candidates, cfg)
		correct := decision.Status == testCase.Expected.Status &&
			(testCase.Expected.Status != StatusMatched || decision.SelectedCandidateID == testCase.Expected.CandidateID)
		top1CorrectForCase := testCase.Expected.CandidateID != "" && decision.TopCandidateID == testCase.Expected.CandidateID
		if testCase.Expected.CandidateID != "" {
			top1Total++
			if top1CorrectForCase {
				top1Correct++
			}
		}

		status := string(decision.Status)
		expectedStatus := string(testCase.Expected.Status)
		summary.StatusCounts[status]++
		summary.ExpectedStatusCount[expectedStatus]++
		if summary.StatusConfusion[expectedStatus] == nil {
			summary.StatusConfusion[expectedStatus] = map[string]int{}
		}
		summary.StatusConfusion[expectedStatus][status]++
		if decision.Alert != nil {
			summary.AlertReasons[decision.Alert.Reason]++
			if decision.Alert.Reason == "top_score_tie" {
				summary.TieAlerts++
			}
		}

		if testCase.Expected.Status == StatusMatched {
			summary.ExpectedMatched++
		}
		if decision.Status == StatusMatched {
			summary.PredictedMatched++
			if correct {
				summary.CorrectAutoMatches++
			} else {
				summary.FalseMerges++
			}
		}
		if testCase.Expected.Status == StatusMatched && !correct {
			summary.MissedMatches++
		}
		if decision.Status == StatusAmbiguous {
			summary.AmbiguousCases++
		}

		results = append(results, CaseResult{
			ID:          testCase.ID,
			Expected:    testCase.Expected,
			Decision:    decision,
			Correct:     correct,
			Top1Correct: top1CorrectForCase,
		})
	}

	summary.AutoMatchPrecision = ratio(summary.CorrectAutoMatches, summary.PredictedMatched)
	summary.AutoMatchRecall = ratio(summary.CorrectAutoMatches, summary.ExpectedMatched)
	summary.AutoMatchF1 = f1(summary.AutoMatchPrecision, summary.AutoMatchRecall)
	summary.Top1Accuracy = ratio(top1Correct, top1Total)
	summary.AmbiguousRate = ratio(summary.AmbiguousCases, summary.TotalCases)

	return Report{
		SchemaVersion: AnalysisSchemaVersion,
		Config:        cfg,
		Summary:       summary,
		Cases:         results,
	}
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func f1(precision, recall float64) float64 {
	if precision+recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}
