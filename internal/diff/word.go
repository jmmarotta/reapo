package diff

import (
	"regexp"
	"strings"
)

// WordDiff represents a word-level difference
type WordDiff struct {
	OldWords []Word
	NewWords []Word
	Changes  []WordChange
}

// Word represents a single word or token
type Word struct {
	Text     string
	IsSpace  bool
	Position int
}

// WordChange represents a change at the word level
type WordChange struct {
	Type     ChangeType
	OldWord  *Word
	NewWord  *Word
	OldIndex int
	NewIndex int
}

// ComputeWordDiff calculates word-level differences between two strings
func ComputeWordDiff(oldLine, newLine string) *WordDiff {
	oldWords := tokenizeLine(oldLine)
	newWords := tokenizeLine(newLine)

	// Use LCS algorithm to find word-level changes
	changes := computeWordLCS(oldWords, newWords)

	return &WordDiff{
		OldWords: oldWords,
		NewWords: newWords,
		Changes:  changes,
	}
}

// tokenizeLine breaks a line into words/tokens
func tokenizeLine(line string) []Word {
	if line == "" {
		return []Word{}
	}

	var words []Word
	position := 0

	// Use regex to split on word boundaries while preserving whitespace
	// This pattern matches: words, numbers, punctuation, or whitespace
	pattern := regexp.MustCompile(`(\w+|[^\w\s]+|\s+)`)
	matches := pattern.FindAllString(line, -1)

	for _, match := range matches {
		isSpace := strings.TrimSpace(match) == ""
		words = append(words, Word{
			Text:     match,
			IsSpace:  isSpace,
			Position: position,
		})
		position += len(match)
	}

	return words
}

// computeWordLCS computes the longest common subsequence for words
func computeWordLCS(oldWords, newWords []Word) []WordChange {
	n := len(oldWords)
	m := len(newWords)

	// Create LCS table
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}

	// Fill LCS table
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if oldWords[i-1].Text == newWords[j-1].Text {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				lcs[i][j] = maxInt(lcs[i-1][j], lcs[i][j-1])
			}
		}
	}

	// Backtrack to find changes
	var changes []WordChange
	i, j := n, m

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldWords[i-1].Text == newWords[j-1].Text {
			// Words match
			changes = append([]WordChange{{
				Type:     Equal,
				OldWord:  &oldWords[i-1],
				NewWord:  &newWords[j-1],
				OldIndex: i - 1,
				NewIndex: j - 1,
			}}, changes...)
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			// Word added
			changes = append([]WordChange{{
				Type:     Insert,
				NewWord:  &newWords[j-1],
				NewIndex: j - 1,
				OldIndex: i,
			}}, changes...)
			j--
		} else if i > 0 {
			// Word deleted
			changes = append([]WordChange{{
				Type:     Delete,
				OldWord:  &oldWords[i-1],
				OldIndex: i - 1,
				NewIndex: j,
			}}, changes...)
			i--
		}
	}

	return changes
}

// HighlightWordChanges applies word-level highlighting to a line
func HighlightWordChanges(line string, changes []WordChange, isOldLine bool) string {
	if len(changes) == 0 {
		return line
	}

	var result strings.Builder
	lastPos := 0

	for _, change := range changes {
		var word *Word
		if isOldLine && change.OldWord != nil {
			word = change.OldWord
		} else if !isOldLine && change.NewWord != nil {
			word = change.NewWord
		} else {
			continue
		}

		// Add text before this word
		if word.Position > lastPos {
			result.WriteString(line[lastPos:word.Position])
		}

		// Apply highlighting based on change type
		switch change.Type {
		case Delete:
			if isOldLine {
				result.WriteString(GetDeletedLineStyle(true).Render(word.Text))
			}
		case Insert:
			if !isOldLine {
				result.WriteString(GetAddedLineStyle(true).Render(word.Text))
			}
		default:
			result.WriteString(word.Text)
		}

		lastPos = word.Position + len(word.Text)
	}

	// Add remaining text
	if lastPos < len(line) {
		result.WriteString(line[lastPos:])
	}

	return result.String()
}

// FindModifiedPairs finds pairs of deleted/added lines that are modifications
func FindModifiedPairs(changes []Change) []ModifiedPair {
	var pairs []ModifiedPair

	for i := 0; i < len(changes)-1; i++ {
		if changes[i].Type == Delete && changes[i+1].Type == Insert {
			// Check if lines are similar enough to be considered a modification
			similarity := calculateSimilarity(
				strings.Join(changes[i].OldLines, " "),
				strings.Join(changes[i+1].NewLines, " "),
			)

			if similarity > 0.3 { // 30% similarity threshold
				pairs = append(pairs, ModifiedPair{
					OldChange:  changes[i],
					NewChange:  changes[i+1],
					Similarity: similarity,
				})
			}
		}
	}

	return pairs
}

// ModifiedPair represents a pair of delete/insert that forms a modification
type ModifiedPair struct {
	OldChange  Change
	NewChange  Change
	Similarity float64
}

// calculateSimilarity calculates similarity between two strings (0.0 to 1.0)
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// Use Levenshtein distance
	distance := levenshteinDistance(s1, s2)
	maxLen := maxInt(len(s1), len(s2))

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	len1 := len(s1)
	len2 := len(s2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create distance matrix
	dist := make([][]int, len1+1)
	for i := range dist {
		dist[i] = make([]int, len2+1)
	}

	// Initialize first column and row
	for i := 0; i <= len1; i++ {
		dist[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dist[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			dist[i][j] = minInt(
				dist[i-1][j]+1, // deletion
				minInt(
					dist[i][j-1]+1,      // insertion
					dist[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return dist[len1][len2]
}

// Helper functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
