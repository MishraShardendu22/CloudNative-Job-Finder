package bm25

import "math"

type Document struct {
	ID     string
	Tokens []string
}

func Rank(query []string, docs []Document, k1, b float64) map[string]float64 {
	scores := make(map[string]float64, len(docs))
	if len(query) == 0 || len(docs) == 0 {
		return scores
	}

	if k1 <= 0 {
		k1 = 1.5
	}
	if b <= 0 || b > 1 {
		b = 0.75
	}

	docFreq := map[string]int{}
	docLengths := map[string]float64{}
	termFreq := map[string]map[string]float64{}

	totalLen := 0.0
	for _, doc := range docs {
		tf := map[string]float64{}
		seen := map[string]struct{}{}
		for _, token := range doc.Tokens {
			tf[token]++
			if _, ok := seen[token]; !ok {
				docFreq[token]++
				seen[token] = struct{}{}
			}
		}
		length := float64(len(doc.Tokens))
		docLengths[doc.ID] = length
		termFreq[doc.ID] = tf
		totalLen += length
	}

	avgDocLen := totalLen / float64(len(docs))
	if avgDocLen == 0 {
		return scores
	}

	for _, doc := range docs {
		score := 0.0
		for _, term := range query {
			nq := float64(docFreq[term])
			if nq == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(docs))-nq+0.5)/(nq+0.5))
			tf := termFreq[doc.ID][term]
			if tf == 0 {
				continue
			}
			numerator := tf * (k1 + 1)
			denominator := tf + k1*(1-b+b*docLengths[doc.ID]/avgDocLen)
			score += idf * numerator / denominator
		}
		scores[doc.ID] = score
	}

	return scores
}
