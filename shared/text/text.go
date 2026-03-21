package text

import (
	"regexp"
	"sort"
	"strings"
)

var tokenPattern = regexp.MustCompile(`[a-zA-Z0-9+#]+`)
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "to": {}, "for": {}, "of": {}, "in": {}, "on": {}, "at": {},
	"is": {}, "are": {}, "be": {}, "as": {}, "with": {}, "by": {}, "from": {}, "this": {}, "that": {}, "you": {},
	"your": {}, "our": {}, "we": {}, "it": {}, "will": {}, "can": {}, "have": {}, "has": {}, "about": {},
	"job": {}, "role": {}, "team": {}, "work": {}, "experience": {}, "years": {},
}

func StripHTML(input string) string {
	return htmlTagPattern.ReplaceAllString(input, " ")
}

func Tokenize(input string) []string {
	raw := tokenPattern.FindAllString(strings.ToLower(input), -1)
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		if len(token) < 2 {
			continue
		}
		if _, isStopWord := stopWords[token]; isStopWord {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func Unique(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	output := make([]string, 0, len(input))
	for _, item := range input {
		normalized := strings.TrimSpace(strings.ToLower(item))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		output = append(output, normalized)
	}
	return output
}

func ExtractTopKeywords(input string, limit int) []string {
	freq := map[string]int{}
	for _, token := range Tokenize(input) {
		freq[token]++
	}
	type pair struct {
		Word  string
		Count int
	}
	pairs := make([]pair, 0, len(freq))
	for word, count := range freq {
		pairs = append(pairs, pair{Word: word, Count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Word < pairs[j].Word
		}
		return pairs[i].Count > pairs[j].Count
	})
	if limit > len(pairs) || limit <= 0 {
		limit = len(pairs)
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, pairs[i].Word)
	}
	return result
}
