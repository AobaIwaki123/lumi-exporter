// Package matching ranks event candidates and emits explainable review decisions.
package matching

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	specEvent "github.com/AobaIwaki123/lumi-spec/event"
	specMatch "github.com/AobaIwaki123/lumi-spec/match"
	"golang.org/x/text/unicode/norm"
)

// Status is the decision made for a query event and its candidate set.
type Status = specMatch.Status

const (
	StatusMatched   = specMatch.StatusMatched
	StatusAmbiguous = specMatch.StatusAmbiguous
	StatusUnmatched = specMatch.StatusUnmatched
)

// Event is the shared lumi-spec event candidate contract.
type Event = specEvent.Candidate

// Weights controls the fixed score contribution of each feature.
type Weights struct {
	Title      float64 `json:"title"`
	Date       float64 `json:"date"`
	Venue      float64 `json:"venue"`
	Performers float64 `json:"performers"`
}

// Config controls the relative ranking and abstention behavior.
type Config struct {
	MinScore   float64 `json:"min_score"`
	MinMargin  float64 `json:"min_margin"`
	TieEpsilon float64 `json:"tie_epsilon"`
	Weights    Weights `json:"weights"`
}

// DefaultConfig is intentionally conservative: a loose title comparison is
// allowed, but an insufficient score margin never becomes an auto-merge.
func DefaultConfig() Config {
	return Config{
		MinScore:   0.55,
		MinMargin:  0.08,
		TieEpsilon: 1e-9,
		Weights: Weights{
			Title:      0.45,
			Date:       0.30,
			Venue:      0.15,
			Performers: 0.10,
		},
	}
}

// FeatureScores exposes the evidence used for a candidate score.
type FeatureScores struct {
	Title           float64 `json:"title"`
	Date            float64 `json:"date"`
	Venue           float64 `json:"venue"`
	Performers      float64 `json:"performers"`
	AvailableWeight float64 `json:"available_weight"`
	Coverage        float64 `json:"coverage"`
}

// ScoredCandidate keeps the candidate and all intermediate values for review.
type ScoredCandidate struct {
	CandidateID string        `json:"candidate_id"`
	Score       float64       `json:"score"`
	Features    FeatureScores `json:"features"`
	Event       Event         `json:"-"`
}

// Alert is emitted whenever the matcher abstains and a human/code review is
// useful. In particular, top_score_tie is a required signal for rule review.
type Alert struct {
	Type       string   `json:"type"`
	Reason     string   `json:"reason"`
	TopScore   float64  `json:"top_score"`
	Candidates []string `json:"candidates"`
}

// Decision is the complete, explainable result for one query event.
type Decision struct {
	SchemaVersion       string            `json:"schema_version"`
	QueryID             string            `json:"query_id"`
	Status              Status            `json:"status"`
	SelectedCandidateID string            `json:"selected_candidate_id,omitempty"`
	TopCandidateID      string            `json:"top_candidate_id,omitempty"`
	TopScore            float64           `json:"top_score"`
	SecondScore         float64           `json:"second_score"`
	ScoreMargin         float64           `json:"score_margin"`
	Candidates          []ScoredCandidate `json:"candidates"`
	Alert               *Alert            `json:"alert,omitempty"`
}

// Match ranks all supplied candidates. Candidate generation is intentionally
// outside this PoC so score behavior can be evaluated independently.
func Match(query Event, candidates []Event, cfg Config) Decision {
	scored := make([]ScoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		features := scoreFeatures(query, candidate, cfg.Weights)
		scored = append(scored, ScoredCandidate{
			CandidateID: candidate.ID,
			Score:       weightedScore(features, cfg.Weights),
			Features:    features,
			Event:       candidate,
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Event.ID < scored[j].Event.ID
		}
		return scored[i].Score > scored[j].Score
	})

	decision := Decision{
		SchemaVersion: specMatch.DecisionSchemaVersion,
		QueryID:       query.ID,
		Status:        StatusUnmatched,
		Candidates:    scored,
	}
	if len(scored) == 0 {
		decision.Alert = reviewAlert("no_candidates", 0, nil)
		return decision
	}
	decision.TopCandidateID = scored[0].Event.ID
	decision.TopScore = scored[0].Score
	if len(scored) > 1 {
		decision.SecondScore = scored[1].Score
		decision.ScoreMargin = scored[0].Score - scored[1].Score
	}

	if decision.TopScore < cfg.MinScore {
		decision.Alert = reviewAlert("top_score_below_minimum", decision.TopScore, scored[:limit(2, len(scored))])
		return decision
	}
	if len(scored) > 1 && math.Abs(decision.ScoreMargin) <= cfg.TieEpsilon {
		decision.Alert = reviewAlert("top_score_tie", decision.TopScore, scored[:limit(2, len(scored))])
		decision.Status = StatusAmbiguous
		return decision
	}
	if len(scored) > 1 && decision.ScoreMargin < cfg.MinMargin {
		decision.Alert = reviewAlert("score_margin_below_minimum", decision.TopScore, scored[:limit(2, len(scored))])
		decision.Status = StatusAmbiguous
		return decision
	}

	decision.Status = StatusMatched
	decision.SelectedCandidateID = decision.TopCandidateID
	return decision
}

func scoreFeatures(query, candidate Event, weights Weights) FeatureScores {
	features := FeatureScores{}
	availableWeight := 0.0
	if score, ok := textSimilarity(query.Title, candidate.Title); ok {
		features.Title = score
		availableWeight += weights.Title
	}
	if score, ok := dateSimilarity(query.LocalDate, candidate.LocalDate); ok {
		features.Date = score
		availableWeight += weights.Date
	}
	if score, ok := textSimilarity(query.Venue, candidate.Venue); ok {
		features.Venue = score
		availableWeight += weights.Venue
	}
	if score, ok := artistSimilarity(query.Performers, candidate.Performers); ok {
		features.Performers = score
		availableWeight += weights.Performers
	}
	features.AvailableWeight = availableWeight
	totalWeight := weights.Title + weights.Date + weights.Venue + weights.Performers
	if totalWeight > 0 {
		features.Coverage = availableWeight / totalWeight
	}
	return features
}

func weightedScore(features FeatureScores, weights Weights) float64 {
	return features.Title*weights.Title +
		features.Date*weights.Date +
		features.Venue*weights.Venue +
		features.Performers*weights.Performers
}

func dateSimilarity(left, right string) (float64, bool) {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return 0, false
	}
	leftDate, leftErr := time.Parse("2006-01-02", left)
	rightDate, rightErr := time.Parse("2006-01-02", right)
	if leftErr != nil || rightErr != nil {
		return 0, false
	}
	days := math.Abs(leftDate.Sub(rightDate).Hours() / 24)
	switch {
	case days == 0:
		return 1, true
	case days <= 1:
		return 0.5, true
	default:
		return 0, true
	}
}

func textSimilarity(left, right string) (float64, bool) {
	left = normalize(left)
	right = normalize(right)
	if left == "" || right == "" {
		return 0, false
	}
	if left == right {
		return 1, true
	}
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	if strings.Contains(left, right) || strings.Contains(right, left) {
		shorter := len(leftRunes)
		longer := len(rightRunes)
		if shorter > longer {
			shorter, longer = longer, shorter
		}
		return 0.9 * float64(shorter) / float64(longer), true
	}
	return diceCoefficient(leftRunes, rightRunes), true
}

func artistSimilarity(left, right []string) (float64, bool) {
	leftSet := normalizedSet(left)
	rightSet := normalizedSet(right)
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0, false
	}
	intersection := 0
	for value := range leftSet {
		if _, ok := rightSet[value]; ok {
			intersection++
		}
	}
	union := len(leftSet) + len(rightSet) - intersection
	if union == 0 {
		return 0, false
	}
	return float64(intersection) / float64(union), true
}

func normalizedSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalize(value); normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
}

func normalize(value string) string {
	value = strings.ToLower(norm.NFKC.String(value))
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func diceCoefficient(left, right []rune) float64 {
	leftBigrams := bigrams(left)
	rightBigrams := bigrams(right)
	if len(leftBigrams) == 0 && len(rightBigrams) == 0 {
		if string(left) == string(right) {
			return 1
		}
		return 0
	}
	intersection := 0
	for value := range leftBigrams {
		if _, ok := rightBigrams[value]; ok {
			intersection++
		}
	}
	return 2 * float64(intersection) / float64(len(leftBigrams)+len(rightBigrams))
}

func bigrams(values []rune) map[string]struct{} {
	result := make(map[string]struct{})
	if len(values) == 1 {
		result[string(values)] = struct{}{}
		return result
	}
	for i := 0; i+1 < len(values); i++ {
		result[string(values[i:i+2])] = struct{}{}
	}
	return result
}

func reviewAlert(reason string, topScore float64, candidates []ScoredCandidate) *Alert {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Event.ID)
	}
	return &Alert{
		Type:       "event_match_review",
		Reason:     reason,
		TopScore:   topScore,
		Candidates: ids,
	}
}

func limit(value, length int) int {
	if value < length {
		return value
	}
	return length
}
