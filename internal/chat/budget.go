package chat

import "strings"

const DefaultCharsPerToken = 4

type TokenEstimator interface {
	EstimateTokens(text string) int
}

type ApproxTokenEstimator struct {
	CharsPerToken int
}

func (e ApproxTokenEstimator) EstimateTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	charsPerToken := e.CharsPerToken
	if charsPerToken <= 0 {
		charsPerToken = DefaultCharsPerToken
	}
	tokens := len([]rune(text)) / charsPerToken
	if len([]rune(text))%charsPerToken != 0 {
		tokens++
	}
	if tokens == 0 {
		return 1
	}
	return tokens
}

type ContextBudget struct {
	ContextWindow      int
	ReservedResponse   int
	ReservedSystem     int
	ReservedChat       int
	SafetyMarginTokens int
}

func (b ContextBudget) AvailableTranscriptTokens() int {
	available := b.ContextWindow - b.ReservedResponse - b.ReservedSystem - b.ReservedChat - b.SafetyMarginTokens
	if available < 0 {
		return 0
	}
	return available
}

func FitTextToTokenBudget(text string, maxTokens int, estimator TokenEstimator) (string, int, bool) {
	if estimator == nil {
		estimator = ApproxTokenEstimator{}
	}
	estimated := estimator.EstimateTokens(text)
	if estimated == 0 || estimated <= maxTokens {
		return text, estimated, false
	}
	if maxTokens <= 0 {
		return "", 0, true
	}
	maxChars := maxTokens * DefaultCharsPerToken
	runes := []rune(text)
	if maxChars > len(runes) {
		maxChars = len(runes)
	}
	fitted := strings.TrimSpace(string(runes[:maxChars]))
	return fitted, estimator.EstimateTokens(fitted), true
}
