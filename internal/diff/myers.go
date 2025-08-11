package diff

import (
	"strings"
)

// Compute calculates the diff between two strings using Myers algorithm
func Compute(oldContent, newContent string, opts Options) *DiffResult {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	
	// Handle empty cases
	if len(oldLines) == 0 && len(newLines) == 0 {
		return &DiffResult{
			Changes:    []Change{},
			Context:    opts.Context,
			OldContent: oldContent,
			NewContent: newContent,
		}
	}
	
	// Run Myers algorithm to find shortest edit script
	script := myers(oldLines, newLines)
	
	// Convert edit script to changes
	changes := scriptToChanges(script, oldLines, newLines)
	
	// Merge consecutive changes of the same type
	changes = mergeChanges(changes)
	
	return &DiffResult{
		Changes:    changes,
		Context:    opts.Context,
		OldContent: oldContent,
		NewContent: newContent,
	}
}

// myers implements the Myers diff algorithm
func myers(oldLines, newLines []string) []Change {
	n := len(oldLines)
	m := len(newLines)
	
	// Handle edge cases
	if n == 0 {
		// All insertions
		return []Change{{
			Type:     Insert,
			OldStart: 1,
			OldLines: []string{},
			NewStart: 1,
			NewLines: newLines,
		}}
	}
	if m == 0 {
		// All deletions
		return []Change{{
			Type:     Delete,
			OldStart: 1,
			OldLines: oldLines,
			NewStart: 1,
			NewLines: []string{},
		}}
	}
	
	// Find longest common subsequence using dynamic programming
	max := n + m
	v := make(map[int]int)
	v[1] = 0
	
	var trace []map[int]int
	
	// Forward search
	for d := 0; d <= max; d++ {
		vCopy := make(map[int]int)
		for k, val := range v {
			vCopy[k] = val
		}
		trace = append(trace, vCopy)
		
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1] < v[k+1]) {
				// Move down (insertion)
				x = v[k+1]
			} else {
				// Move right (deletion)
				x = v[k-1] + 1
			}
			
			y := x - k
			
			// Follow diagonal (matching lines)
			for x < n && y < m && oldLines[x] == newLines[y] {
				x++
				y++
			}
			
			v[k] = x
			
			// Check if we reached the end
			if x >= n && y >= m {
				return buildPath(trace, n, m, oldLines, newLines)
			}
		}
	}
	
	// Shouldn't reach here
	return []Change{}
}

// buildPath reconstructs the edit path from the trace
func buildPath(trace []map[int]int, n, m int, oldLines, newLines []string) []Change {
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
		if prevK == k + 1 {
			// Insertion
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
		} else if prevK == k - 1 {
			// Deletion
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

// scriptToChanges converts an edit script to a list of changes
func scriptToChanges(script []Change, oldLines, newLines []string) []Change {
	if len(script) == 0 {
		// No changes, everything is equal
		return []Change{{
			Type:     Equal,
			OldStart: 1,
			OldLines: oldLines,
			NewStart: 1,
			NewLines: newLines,
		}}
	}
	return script
}

// mergeChanges merges consecutive changes of the same type
func mergeChanges(changes []Change) []Change {
	if len(changes) <= 1 {
		return changes
	}
	
	var merged []Change
	current := changes[0]
	
	for i := 1; i < len(changes); i++ {
		if changes[i].Type == current.Type &&
			changes[i].OldStart == current.OldStart+len(current.OldLines) &&
			changes[i].NewStart == current.NewStart+len(current.NewLines) {
			// Merge with current
			current.OldLines = append(current.OldLines, changes[i].OldLines...)
			current.NewLines = append(current.NewLines, changes[i].NewLines...)
		} else {
			// Start new change
			merged = append(merged, current)
			current = changes[i]
		}
	}
	merged = append(merged, current)
	
	return merged
}

// ComputeSimple computes a simple diff for find/replace operations
func ComputeSimple(oldString, newString string) *DiffResult {
	return Compute(oldString, newString, DefaultOptions())
}