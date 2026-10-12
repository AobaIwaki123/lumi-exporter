package matching

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluationFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "evaluation.json")
	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = file.Close() }()

	fixture, err := LoadFixture(file)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	report := Evaluate(fixture, DefaultConfig())

	if report.Summary.TotalCases != 6 {
		t.Fatalf("total cases = %d, want 6", report.Summary.TotalCases)
	}
	if report.Summary.CorrectAutoMatches != 3 {
		t.Fatalf("correct auto matches = %d, want 3", report.Summary.CorrectAutoMatches)
	}
	if report.Summary.FalseMerges != 0 {
		t.Fatalf("false merges = %d, want 0", report.Summary.FalseMerges)
	}
	if report.Summary.TieAlerts != 1 {
		t.Fatalf("tie alerts = %d, want 1", report.Summary.TieAlerts)
	}
	if report.Summary.AutoMatchPrecision != 1 {
		t.Fatalf("precision = %f, want 1", report.Summary.AutoMatchPrecision)
	}
	if report.Summary.AutoMatchRecall != 1 {
		t.Fatalf("recall = %f, want 1", report.Summary.AutoMatchRecall)
	}
}

func TestMatchEmitsTopScoreTieAlert(t *testing.T) {
	query := Event{ID: "query", Title: "合同ライブ", LocalDate: "2026-11-03", Venue: "下北沢ERA"}
	candidates := []Event{
		{ID: "candidate-a", Title: "合同ライブ", LocalDate: "2026-11-03", Venue: "下北沢ERA"},
		{ID: "candidate-b", Title: "合同ライブ", LocalDate: "2026-11-03", Venue: "下北沢ERA"},
	}

	decision := Match(query, candidates, DefaultConfig())
	if decision.Status != StatusAmbiguous {
		t.Fatalf("status = %q, want %q", decision.Status, StatusAmbiguous)
	}
	if decision.Alert == nil || decision.Alert.Reason != "top_score_tie" {
		t.Fatalf("alert = %#v, want top_score_tie", decision.Alert)
	}
	if decision.SelectedCandidateID != "" {
		t.Fatalf("selected candidate = %q, want empty", decision.SelectedCandidateID)
	}
}
