I want to implement the version of the Meyer's diff algorithm that is used in VS Code to show the difference between two files or find and replace strings that we're generating.

I want to implement this in a diff.lua file and I want it to handle the diffing an the rendering of the diff in the TUI if possible. Can we create a vaild "git" type diff output with this and a unified diff that is styled with lipgloss?

Here are some notes on VS Code's implementation:

<vscode-diff-notes>
### The Edit Graph Approach
The algorithm constructs an "edit graph" where:
- **X-axis** represents positions in file A (original)
- **Y-axis** represents positions in file B (modified)
- **Diagonal moves** represent matching lines (no change)
- **Horizontal moves** represent deletions
- **Vertical moves** represent insertions

The algorithm finds the **shortest edit script** - the minimum number of operations to transform file A into file B.

### Longest Common Subsequence (LCS)
At its core, the diff algorithm solves the LCS problem:
1. Build a matrix comparing every line of file A with every line of file B
2. Find the longest sequence of matching lines that appear in the same relative order
3. Everything not in the LCS becomes an insertion or deletion

## Line-by-Line Processing

### Preprocessing Steps
1. **Tokenization**: Files are split into lines (using different line ending detection)
2. **Hashing**: Each line gets a hash for fast comparison
3. **Similarity scoring**: Lines are compared for partial matches

### Hash-Based Optimization
Instead of comparing full line strings repeatedly:
```
Line: "    function foo() {"
Hash: 0x7a3b9c1d
```
This allows O(1) equality checks instead of O(n) string comparisons.

## Advanced Features

### Word-Level Diffing
When lines are identified as "changed" (not pure insertions/deletions):
1. The algorithm runs recursively at the **character/word level**
2. Uses a similar Myers algorithm but on word boundaries
3. Highlights specific words/characters that changed within the line

### Move Detection
VS Code includes sophisticated move detection:
1. **Block similarity analysis**: Identifies when code blocks have moved
2. **Content fingerprinting**: Uses hash signatures to detect relocated content
3. **Threshold-based matching**: Only considers moves above a certain similarity score

### Whitespace Handling
- **Show all whitespace changes** (default)

## Performance Optimizations

### Space Complexity Reduction
Classic diff algorithms are O(n²) space, but VS Code uses:
- **Linear space optimization**: Only keeps current and previous rows of the DP matrix
- **Divide and conquer**: Recursively splits large files into smaller chunks
- **Early termination**: Stops when differences become too numerous

### Lazy Computation
```typescript
// Simplified concept
class DiffComputer {
  computeOnDemand(startLine: number, endLine: number) {
    // Only compute diff for visible viewport
    // Cache results for future requests
  }
}
```

### Memory Management
- **Streaming for large files**: Doesn't load entire files into memory
- **Incremental updates**: Recomputes only changed regions when files update
- **Result caching**: Caches diff results until files change

## Internal Data Structures

### Change Objects
```typescript
interface LineChange {
  originalStartLineNumber: number;
  originalEndLineNumber: number;
  modifiedStartLineNumber: number;
  modifiedEndLineNumber: number;
  charChanges?: CharacterChange[];
}
```

### Edit Operations
The algorithm produces a sequence of edit operations:
- **EQUAL**: Lines match exactly
- **DELETE**: Line exists in original but not modified
- **INSERT**: Line exists in modified but not original
- **REPLACE**: Line changed (combination of delete + insert)

## Integration with Monaco Editor

VS Code's diff view is built on the Monaco Editor engine:

### Rendering Pipeline
1. **Diff computation** produces change objects
2. **Layout engine** determines side-by-side or inline positioning
3. **Syntax highlighting** applied independently to each side
4. **Decoration rendering** adds background colors, gutter indicators
5. **Viewport virtualization** for performance with large files
</vscode-diff-notes>
