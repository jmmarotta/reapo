package diff

import (
	"hash/fnv"
	"strings"
)

// ComputeOptimized computes diff with optimizations like line hashing
func ComputeOptimized(oldContent, newContent string, opts Options) *DiffResult {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Hash lines for faster comparison
	oldHashes := hashLines(oldLines)
	newHashes := hashLines(newLines)

	// Run optimized Myers algorithm
	script := myersOptimized(oldHashes, newHashes, oldLines, newLines)

	// Convert edit script to changes
	changes := scriptToChanges(script, oldLines, newLines)

	// Merge consecutive changes
	changes = mergeChanges(changes)

	return &DiffResult{
		Changes:    changes,
		Context:    opts.Context,
		OldContent: oldContent,
		NewContent: newContent,
	}
}

// hashLines creates hashes for each line
func hashLines(lines []string) []uint64 {
	hashes := make([]uint64, len(lines))
	for i, line := range lines {
		hashes[i] = hashLine(line)
	}
	return hashes
}

// hashLine computes a hash for a single line
func hashLine(line string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(line))
	return h.Sum64()
}

// myersOptimized implements Myers algorithm with line hashing
func myersOptimized(oldHashes, newHashes []uint64, oldLines, newLines []string) []Change {
	n := len(oldHashes)
	m := len(newHashes)

	// Handle edge cases
	if n == 0 {
		return []Change{{
			Type:     Insert,
			OldStart: 1,
			OldLines: []string{},
			NewStart: 1,
			NewLines: newLines,
		}}
	}
	if m == 0 {
		return []Change{{
			Type:     Delete,
			OldStart: 1,
			OldLines: oldLines,
			NewStart: 1,
			NewLines: []string{},
		}}
	}

	max := n + m
	v := make(map[int]int)
	v[1] = 0

	var trace []map[int]int

	for d := 0; d <= max; d++ {
		vCopy := make(map[int]int)
		for k, val := range v {
			vCopy[k] = val
		}
		trace = append(trace, vCopy)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1] < v[k+1]) {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}

			y := x - k

			// Follow diagonal using hash comparison
			for x < n && y < m && oldHashes[x] == newHashes[y] {
				x++
				y++
			}

			v[k] = x

			if x >= n && y >= m {
				return buildPathOptimized(trace, n, m, oldLines, newLines, oldHashes, newHashes)
			}
		}
	}

	return []Change{}
}

// buildPathOptimized reconstructs the edit path using hashes
func buildPathOptimized(trace []map[int]int, n, m int, oldLines, newLines []string, oldHashes, newHashes []uint64) []Change {
	var changes []Change

	// Handle empty trace
	if len(trace) == 0 {
		return changes
	}

	x := n
	y := m

	for d := len(trace) - 1; d >= 0 && (x > 0 || y > 0); d-- {
		v := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK]
		prevY := prevX - prevK

		// Add matching lines
		for x > prevX && y > prevY && x > 0 && y > 0 {
			x--
			y--
			if x >= 0 && x < len(oldLines) && y >= 0 && y < len(newLines) {
				changes = append([]Change{{
					Type:     Equal,
					OldStart: x + 1,
					OldLines: []string{oldLines[x]},
					NewStart: y + 1,
					NewLines: []string{newLines[y]},
				}}, changes...)
			}
		}

		// Add insertion or deletion
		if prevK == k+1 {
			y--
			if y >= 0 && y < len(newLines) {
				changes = append([]Change{{
					Type:     Insert,
					OldStart: x + 1,
					OldLines: []string{},
					NewStart: y + 1,
					NewLines: []string{newLines[y]},
				}}, changes...)
			}
		} else if prevK == k-1 {
			x--
			if x >= 0 && x < len(oldLines) {
				changes = append([]Change{{
					Type:     Delete,
					OldStart: x + 1,
					OldLines: []string{oldLines[x]},
					NewStart: y + 1,
					NewLines: []string{},
				}}, changes...)
			}
		}

		x = prevX
		y = prevY
	}

	return changes
}

// FindLCS finds the longest common subsequence between two line arrays
func FindLCS(oldLines, newLines []string) []string {
	oldHashes := hashLines(oldLines)
	newHashes := hashLines(newLines)

	n := len(oldLines)
	m := len(newLines)

	// Dynamic programming table
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	// Fill the table
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if oldHashes[i-1] == newHashes[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Reconstruct LCS
	var lcs []string
	i, j := n, m
	for i > 0 && j > 0 {
		if oldHashes[i-1] == newHashes[j-1] {
			lcs = append([]string{oldLines[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
