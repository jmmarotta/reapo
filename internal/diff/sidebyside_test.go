package diff

import (
	"strings"
	"testing"
)

func TestFormatSideBySide(t *testing.T) {
	oldContent := `func hello() {
	fmt.Println("Hello, World!")
	return nil
}`

	newContent := `func hello() {
	fmt.Println("Hello, Go!")
	fmt.Println("Welcome!")
	return nil
}`

	result := ComputeSimple(oldContent, newContent)

	opts := SideBySideOptions{
		ColumnWidth:     30,
		ShowLineNumbers: true,
		WordDiff:        true,
		ShowFileHeader:  true,
		FilePath:        "test.go",
	}

	output := FormatSideBySide(result, opts)

	// Check that output contains expected elements
	if !strings.Contains(output, "test.go") {
		t.Error("Output should contain file path")
	}

	if !strings.Contains(output, "Hello, World!") {
		t.Error("Output should contain old content")
	}

	if !strings.Contains(output, "Hello, Go!") {
		t.Error("Output should contain new content")
	}

	if !strings.Contains(output, "Welcome!") {
		t.Error("Output should contain added line")
	}

	// Check for proper formatting elements
	if !strings.Contains(output, "│") {
		t.Error("Output should contain column separators")
	}

	t.Logf("Side-by-side diff output:\n%s", output)
}

func TestWordDiff(t *testing.T) {
	oldLine := "The quick brown fox jumps"
	newLine := "The slow brown cat jumps"

	diff := ComputeWordDiff(oldLine, newLine)

	if len(diff.Changes) == 0 {
		t.Error("Should detect word changes")
	}

	// Check that we detected the changes
	hasQuickDelete := false
	hasSlowInsert := false
	hasFoxDelete := false
	hasCatInsert := false

	for _, change := range diff.Changes {
		if change.Type == Delete && change.OldWord != nil && change.OldWord.Text == "quick" {
			hasQuickDelete = true
		}
		if change.Type == Insert && change.NewWord != nil && change.NewWord.Text == "slow" {
			hasSlowInsert = true
		}
		if change.Type == Delete && change.OldWord != nil && change.OldWord.Text == "fox" {
			hasFoxDelete = true
		}
		if change.Type == Insert && change.NewWord != nil && change.NewWord.Text == "cat" {
			hasCatInsert = true
		}
	}

	if !hasQuickDelete || !hasSlowInsert || !hasFoxDelete || !hasCatInsert {
		t.Error("Should detect all word changes correctly")
	}
}
