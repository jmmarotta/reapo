package diff

// ChangeType represents the type of change in a diff
type ChangeType int

const (
	// Equal indicates lines are the same
	Equal ChangeType = iota
	// Insert indicates lines were added
	Insert
	// Delete indicates lines were removed
	Delete
)

// Change represents a single change in the diff
type Change struct {
	Type      ChangeType
	OldStart  int      // Starting line in old content (1-based)
	OldLines  []string // Lines from old content
	NewStart  int      // Starting line in new content (1-based)
	NewLines  []string // Lines from new content
}

// DiffResult contains the complete diff information
type DiffResult struct {
	Changes    []Change
	Context    int    // Number of context lines to show
	OldContent string // Original content
	NewContent string // Modified content
}

// Options configures diff behavior
type Options struct {
	Context         int  // Number of context lines (default 3)
	IgnoreWhitespace bool // Whether to ignore whitespace changes
	WordLevel       bool // Enable word-level diffing for changes
}

// DefaultOptions returns default diff options
func DefaultOptions() Options {
	return Options{
		Context:         3,
		IgnoreWhitespace: false,
		WordLevel:       false,
	}
}

// LineHash represents a hashed line for fast comparison
type LineHash struct {
	Hash    uint64
	Content string
	LineNum int
}

// EditNode represents a node in the edit graph for Myers algorithm
type EditNode struct {
	X    int // Position in old content
	Y    int // Position in new content
	Prev *EditNode
}

// Snake represents a diagonal move in the edit graph (matching lines)
type Snake struct {
	StartX int
	StartY int
	EndX   int
	EndY   int
}