package tortise_go

// Tests for Tortise File Format Specification v0.2
// - Added support testing for additional symbol delimiters (::, ---, +++, ~~~, @@)  
// - Added tests for emoji/Unicode delimiter parsing and collision detection
// - Implemented Unicode delimiter support per spec v0.2 - any Unicode character
//   except ASCII space (0x20), tab (0x09), LF (0x0A), or CR (0x0D) is allowed
// - Verified existing ASCII delimiter functionality remains intact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseSimpleTortiseFile(t *testing.T) {
	input := `> file1.txt
hello world

> dir/file2.go
package main

func main() {
    println("hello")
}
`
	
	doc, err := ParseTortiseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTortiseFile failed: %v", err)
	}
	
	if doc.Delimiter != ">" {
		t.Errorf("Expected delimiter '>', got '%s'", doc.Delimiter)
	}
	
	if len(doc.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(doc.Files))
	}
	
	if doc.Files[0].Path != "file1.txt" {
		t.Errorf("Expected path 'file1.txt', got '%s'", doc.Files[0].Path)
	}
	
	if doc.Files[0].Content != "hello world\n\n" {
		t.Errorf("Expected content 'hello world\\n\\n', got %q", doc.Files[0].Content)
	}
	
	if doc.Files[1].Path != "dir/file2.go" {
		t.Errorf("Expected path 'dir/file2.go', got '%s'", doc.Files[1].Path)
	}
	
	expectedContent := "package main\n\nfunc main() {\n    println(\"hello\")\n}\n"
	if doc.Files[1].Content != expectedContent {
		t.Errorf("Content mismatch.\nExpected: %q\nGot: %q", expectedContent, doc.Files[1].Content)
	}
}

func TestParseWithDifferentDelimiter(t *testing.T) {
	input := `=== file1.txt
content with > character

=== file2.txt
more content
`
	
	doc, err := ParseTortiseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTortiseFile failed: %v", err)
	}
	
	if doc.Delimiter != "===" {
		t.Errorf("Expected delimiter '===', got '%s'", doc.Delimiter)
	}
	
	if len(doc.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(doc.Files))
	}
	
	if doc.Files[0].Content != "content with > character\n\n" {
		t.Errorf("Expected content with > character, got %q", doc.Files[0].Content)
	}
}

func TestParseEmptyFile(t *testing.T) {
	input := ""
	
	doc, err := ParseTortiseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTortiseFile failed: %v", err)
	}
	
	if len(doc.Files) != 0 {
		t.Errorf("Expected 0 files for empty input, got %d", len(doc.Files))
	}
}

func TestParseWithBlankLines(t *testing.T) {
	input := `

> file1.txt
content


> file2.txt

another line

`
	
	doc, err := ParseTortiseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTortiseFile failed: %v", err)
	}
	
	if len(doc.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(doc.Files))
	}
	
	if doc.Files[0].Content != "content\n\n\n" {
		t.Errorf("Expected 'content\\n\\n\\n', got %q", doc.Files[0].Content)
	}
	
	if doc.Files[1].Content != "\nanother line\n\n" {
		t.Errorf("Expected blank lines to be preserved, got %q", doc.Files[1].Content)
	}
}

func TestParseInvalidPath(t *testing.T) {
	tests := []string{
		"> /absolute/path\ncontent\n",
		"> ../parent/path\ncontent\n",
		"> .\ncontent\n",
		"> \ncontent\n",
	}
	
	for _, input := range tests {
		_, err := ParseTortiseFile(strings.NewReader(input))
		if err == nil {
			t.Errorf("Expected error for invalid path in input: %q", input)
		}
	}
}

func TestParseDuplicatePath(t *testing.T) {
	input := `> file1.txt
content1

> file1.txt
content2
`
	
	_, err := ParseTortiseFile(strings.NewReader(input))
	if err == nil {
		t.Error("Expected error for duplicate path")
	}
}

func TestParseWithEmojiDelimiters(t *testing.T) {
	input := `🐢 src/util.py
a = 1

🐢 hi.py
from src.util import a
print(a)

🐢 config/settings.json
{ "debug": true }
`
	
	doc, err := ParseTortiseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseTortiseFile failed: %v", err)
	}
	
	if doc.Delimiter != "🐢" {
		t.Errorf("Expected delimiter '🐢', got '%s'", doc.Delimiter)
	}
	
	if len(doc.Files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(doc.Files))
	}
	
	expectedFiles := map[string]string{
		"src/util.py":          "a = 1\n\n",
		"hi.py":                "from src.util import a\nprint(a)\n\n",
		"config/settings.json": "{ \"debug\": true }\n",
	}
	
	for i, file := range doc.Files {
		expectedContent, exists := expectedFiles[file.Path]
		if !exists {
			t.Errorf("Unexpected file path: %s", file.Path)
			continue
		}
		
		if file.Content != expectedContent {
			t.Errorf("Content mismatch for file %d (%s).\nExpected: %q\nGot: %q", 
				i, file.Path, expectedContent, file.Content)
		}
	}
}

func TestParseWithUnicodeSymbolDelimiters(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
		input     string
	}{
		{
			name:      "diamond symbols",
			delimiter: "❖❖❖",
			input: `❖❖❖ file1.txt
content with unicode ñoño
❖❖❖ file2.txt
more content 中文
`,
		},
		{
			name:      "math symbols",
			delimiter: "∴",
			input: `∴ math.txt
therefore symbol as delimiter
∴ proof.txt
another mathematical file
`,
		},
		{
			name:      "lambda symbol",
			delimiter: "λ",
			input: `λ functional.hs
map :: (a -> b) -> [a] -> [b]
λ types.hs
data Maybe a = Nothing | Just a
`,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc, err := ParseTortiseFile(strings.NewReader(test.input))
			if err != nil {
				t.Fatalf("ParseTortiseFile failed for %s: %v", test.name, err)
			}
			
			if doc.Delimiter != test.delimiter {
				t.Errorf("Expected delimiter '%s', got '%s'", test.delimiter, doc.Delimiter)
			}
			
			if len(doc.Files) != 2 {
				t.Fatalf("Expected 2 files, got %d", len(doc.Files))
			}
		})
	}
}

func TestWriteTo(t *testing.T) {
	doc := &TortiseDocument{
		Delimiter: ">",
		Files: []TortiseFile{
			{Path: "file1.txt", Content: "hello\n"},
			{Path: "dir/file2.go", Content: "package main\n"},
		},
	}
	
	var buf strings.Builder
	err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	
	expected := `> file1.txt
hello
> dir/file2.go
package main
`
	
	if buf.String() != expected {
		t.Errorf("WriteTo output mismatch.\nExpected: %q\nGot: %q", expected, buf.String())
	}
}

func TestEmojiDelimiterCollisionDetection(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
		content   string
		shouldErr bool
	}{
		{
			name:      "emoji collision detected",
			delimiter: "🐢",
			content:   "🐢 this line conflicts with turtle emoji\nother content\n",
			shouldErr: true,
		},
		{
			name:      "no emoji collision",
			delimiter: "🐢",
			content:   "🚀 this rocket doesn't conflict with turtle\nother content\n",
			shouldErr: false,
		},
		{
			name:      "repeated emoji collision",
			delimiter: "❖❖❖",
			content:   "normal line\n❖❖❖ this conflicts\nmore content\n",
			shouldErr: true,
		},
		{
			name:      "unicode symbol collision",
			delimiter: "∞",
			content:   "∞ infinity symbol conflicts\nmath content\n",
			shouldErr: true,
		},
		{
			name:      "mixed unicode no collision", 
			delimiter: "λ",
			content:   "function definition\n中文 chinese text\nñoño spanish\n",
			shouldErr: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := &TortiseDocument{
				Delimiter: test.delimiter,
				Files: []TortiseFile{
					{Path: "test.txt", Content: test.content},
				},
			}
			
			var buf strings.Builder
			err := doc.WriteTo(&buf)
			
			if test.shouldErr {
				if err == nil {
					t.Errorf("Expected collision error for delimiter %q with content %q", 
						test.delimiter, test.content)
				} else if !strings.Contains(err.Error(), "conflicts with content") {
					t.Errorf("Expected collision error message, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for delimiter %q: %v", test.delimiter, err)
				}
			}
		})
	}
}

func TestWriteToWithContentCollision(t *testing.T) {
	doc := &TortiseDocument{
		Delimiter: ">",
		Files: []TortiseFile{
			{Path: "file1.txt", Content: "> this starts with delimiter\n"},
		},
	}
	
	var buf strings.Builder
	err := doc.WriteTo(&buf)
	if err == nil {
		t.Error("Expected error for content collision")
	}
	
	if !strings.Contains(err.Error(), "conflicts with content") {
		t.Errorf("Expected helpful collision error message, got: %v", err)
	}
	
	if !strings.Contains(err.Error(), "auto-generated delimiter") {
		t.Errorf("Expected suggestion for auto-generated delimiter, got: %v", err)
	}
}

func TestEmojiDelimiterRoundTrip(t *testing.T) {
	// Test that files written with emoji delimiters can be read back correctly
	original := &TortiseDocument{
		Delimiter: "🐢",
		Files: []TortiseFile{
			{Path: "main.py", Content: "print('Hello 🌍')\n"},
			{Path: "config.json", Content: "{\n  \"emoji\": \"🚀\",\n  \"unicode\": \"中文\"\n}\n"},
			{Path: "math.txt", Content: "∞ + 1 = ∞\nλx.x + 1\n"},
		},
	}
	
	// Write to string
	var buf strings.Builder
	err := original.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	
	// Parse back
	parsed, err := ParseTortiseFile(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("ParseTortiseFile failed: %v", err)
	}
	
	// Verify delimiter
	if parsed.Delimiter != "🐢" {
		t.Errorf("Delimiter mismatch. Expected '🐢', got '%s'", parsed.Delimiter)
	}
	
	// Verify files
	if len(parsed.Files) != len(original.Files) {
		t.Fatalf("File count mismatch. Expected %d, got %d", 
			len(original.Files), len(parsed.Files))
	}
	
	for i, originalFile := range original.Files {
		parsedFile := parsed.Files[i]
		if parsedFile.Path != originalFile.Path {
			t.Errorf("Path mismatch at index %d. Expected '%s', got '%s'", 
				i, originalFile.Path, parsedFile.Path)
		}
		if parsedFile.Content != originalFile.Content {
			t.Errorf("Content mismatch for %s.\nExpected: %q\nGot: %q", 
				originalFile.Path, originalFile.Content, parsedFile.Content)
		}
	}
}

func TestDirectoryTreeRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	
	files := map[string]string{
		"file1.txt":        "hello world\n",
		"dir/file2.go":     "package main\n\nfunc main() {}\n",
		"dir/subdir/file3": "nested content\n",
	}
	
	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}
	
	doc, err := ReadDirectoryTree(tempDir)
	if err != nil {
		t.Fatalf("ReadDirectoryTree failed: %v", err)
	}
	
	if len(doc.Files) != len(files) {
		t.Fatalf("Expected %d files, got %d", len(files), len(doc.Files))
	}
	
	outputDir := t.TempDir()
	err = doc.WriteToDirectory(outputDir)
	if err != nil {
		t.Fatalf("WriteToDirectory failed: %v", err)
	}
	
	for path, expectedContent := range files {
		fullPath := filepath.Join(outputDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Failed to read output file %s: %v", path, err)
		}
		
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s.\nExpected: %q\nGot: %q", path, expectedContent, string(content))
		}
	}
}

func TestDelimiterDetection(t *testing.T) {
	tests := []struct {
		line     string
		delim    string
		path     string
		hasError bool
	}{
		{"> file.txt", ">", "file.txt", false},
		{"=== file.txt", "===", "file.txt", false},
		{"*** file.txt", "***", "file.txt", false},
		{"-> file.txt", "->", "file.txt", false},
		{"## file.txt", "##", "file.txt", false},
		// Additional symbol delimiters from spec
		{":: file.txt", "::", "file.txt", false},
		{"--- file.txt", "---", "file.txt", false},
		{"+++ file.txt", "+++", "file.txt", false},
		{"~~~ file.txt", "~~~", "file.txt", false},
		{"@@ file.txt", "@@", "file.txt", false},
		// Emoji/Unicode delimiters (now supported per spec v0.2)
		{"🐢 file.txt", "🐢", "file.txt", false},
		{"❖❖❖ file.txt", "❖❖❖", "file.txt", false},
		{"🚀 src/main.go", "🚀", "src/main.go", false},
		{"⭐⭐ config.json", "⭐⭐", "config.json", false},
		{"🔥🔥🔥 test.py", "🔥🔥🔥", "test.py", false},
		{"∴ math.txt", "∴", "math.txt", false},
		{"∞∞ infinity.md", "∞∞", "infinity.md", false},
		{"λ lambda.hs", "λ", "lambda.hs", false},
		{"αβγ greek.txt", "αβγ", "greek.txt", false},
		{"中文 chinese.txt", "中文", "chinese.txt", false},
		{"file.txt", "", "", true},
		{">", "", "", true},
		{"", "", "", true},
		{"> ", "", "", true},
	}
	
	for _, test := range tests {
		delim, path, err := detectDelimiter(test.line)
		
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for line %q", test.line)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("Unexpected error for line %q: %v", test.line, err)
			continue
		}
		
		if delim != test.delim {
			t.Errorf("Delimiter mismatch for line %q. Expected %q, got %q", test.line, test.delim, delim)
		}
		
		if path != test.path {
			t.Errorf("Path mismatch for line %q. Expected %q, got %q", test.line, test.path, path)
		}
	}
}

func TestValidatePath(t *testing.T) {
	validPaths := []string{
		"file.txt",
		"dir/file.txt",
		"deeply/nested/dir/file.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
	}
	
	for _, path := range validPaths {
		if err := validatePath(path); err != nil {
			t.Errorf("Expected valid path %q to pass validation, got error: %v", path, err)
		}
	}
	
	invalidPaths := []string{
		"",
		".",
		"/absolute/path",
		"../parent",
		"dir/../parent",
		"path/with/../parent",
	}
	
	for _, path := range invalidPaths {
		if err := validatePath(path); err == nil {
			t.Errorf("Expected invalid path %q to fail validation", path)
		}
	}
}

func TestReadFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	files := map[string]string{
		"file1.txt": "content of file1\n",
		"file2.go":  "package main\n\nfunc main() {}\n",
	}
	
	filePaths := []string{}
	for name, content := range files {
		fullPath := filepath.Join(tempDir, name)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		filePaths = append(filePaths, fullPath)
	}
	
	doc, err := ReadFiles(filePaths)
	if err != nil {
		t.Fatalf("ReadFiles failed: %v", err)
	}
	
	if len(doc.Files) != len(files) {
		t.Fatalf("Expected %d files, got %d", len(files), len(doc.Files))
	}
	
	for _, file := range doc.Files {
		expectedContent, exists := files[filepath.Base(file.Path)]
		if !exists {
			t.Errorf("Unexpected file in result: %s", file.Path)
			continue
		}
		
		if file.Content != expectedContent {
			t.Errorf("Content mismatch for %s.\nExpected: %q\nGot: %q", file.Path, expectedContent, file.Content)
		}
	}
}

func TestReadFilesWithDirectory(t *testing.T) {
	tempDir := t.TempDir()
	
	_, err := ReadFiles([]string{tempDir})
	if err == nil {
		t.Error("Expected error when passing directory to ReadFiles")
	}
}

func TestReadFilesNonexistent(t *testing.T) {
	_, err := ReadFiles([]string{"nonexistent.txt"})
	if err == nil {
		t.Error("Expected error when passing nonexistent file to ReadFiles")
	}
}

func TestFindSafeDelimiter(t *testing.T) {
	tests := []struct {
		name        string
		files       []TortiseFile
		expected    string
		description string
	}{
		{
			name: "no conflicts",
			files: []TortiseFile{
				{Path: "file1.txt", Content: "hello world\n"},
				{Path: "file2.txt", Content: "another line\n"},
			},
			expected:    ">",
			description: "should prefer > when no conflicts",
		},
		{
			name: "conflict with single >",
			files: []TortiseFile{
				{Path: "file1.txt", Content: "> this conflicts\nhello world\n"},
			},
			expected:    "=",
			description: "should prefer = when > conflicts (same length, next preference)",
		},
		{
			name: "conflict with > and =",
			files: []TortiseFile{
				{Path: "file1.txt", Content: "> this conflicts\n= also conflicts\n"},
			},
			expected:    "*",
			description: "should prefer * when > and = conflict (same length, next preference)",
		},
		{
			name: "multiple conflicts same length",
			files: []TortiseFile{
				{Path: "file1.txt", Content: "> conflicts\n= also conflicts\n* also conflicts\n"},
			},
			expected:    "-",
			description: "should fall back to - when >, =, * all conflict",
		},
		{
			name: "all single chars conflict",
			files: []TortiseFile{
				{Path: "file1.txt", Content: "> conflicts\n= also conflicts\n* also conflicts\n- also conflicts\n"},
			},
			expected:    ">>",
			description: "should use >> when all single chars conflict",
		},
		{
			name: "prefer shorter length",
			files: []TortiseFile{
				{Path: "file1.txt", Content: ">>> conflicts\n"},
			},
			expected:    ">",
			description: "should prefer single > over longer when no conflict",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := &TortiseDocument{Files: test.files}
			result, err := findSafeDelimiter(doc)
			if err != nil {
				t.Fatalf("findSafeDelimiter failed: %v", err)
			}
			if result != test.expected {
				t.Errorf("%s: expected %q, got %q", test.description, test.expected, result)
			}
		})
	}
}

func TestAutoDelimiterInWriteTo(t *testing.T) {
	doc := &TortiseDocument{
		Files: []TortiseFile{
			{Path: "file1.txt", Content: "> this line conflicts with >\n"},
			{Path: "file2.txt", Content: "normal content\n"},
		},
	}
	
	var buf strings.Builder
	err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	
	output := buf.String()
	if !strings.HasPrefix(output, "= file1.txt\n") {
		t.Errorf("Expected auto-selected delimiter =, got output: %s", output[:20])
	}
}

func TestFindSafeDelimiterNoSolution(t *testing.T) {
	content := ""
	for _, char := range []rune{'>', '=', '*', '-'} {
		for length := 1; length <= 50; length++ {
			delimiter := strings.Repeat(string(char), length)
			content += delimiter + " conflicts\n"
		}
	}
	
	doc := &TortiseDocument{
		Files: []TortiseFile{
			{Path: "impossible.txt", Content: content},
		},
	}
	
	_, err := findSafeDelimiter(doc)
	if err == nil {
		t.Error("Expected error when no safe delimiter can be found")
	}
	
	if !strings.Contains(err.Error(), "unable to find safe delimiter") {
		t.Errorf("Expected 'unable to find safe delimiter' error, got: %v", err)
	}
}

func TestWriteToNoSafeDelimiter(t *testing.T) {
	content := ""
	for _, char := range []rune{'>', '=', '*', '-'} {
		for length := 1; length <= 50; length++ {
			delimiter := strings.Repeat(string(char), length)
			content += delimiter + " conflicts\n"
		}
	}
	
	doc := &TortiseDocument{
		Files: []TortiseFile{
			{Path: "impossible.txt", Content: content},
		},
	}
	
	var buf strings.Builder
	err := doc.WriteTo(&buf)
	if err == nil {
		t.Error("Expected error when no safe delimiter can be found")
	}
}

func TestAutoDiscoveryEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: ">",
		},
		{
			name:     "only whitespace",
			content:  "   \n\t\n   ",
			expected: ">",
		},
		{
			name:     "gt at end of line",
			content:  "some text >\nmore text",
			expected: ">",
		},
		{
			name:     "gt without space",
			content:  ">noSpace\n>alsoNoSpace",
			expected: ">",
		},
		{
			name:     "gt with multiple spaces",
			content:  ">  multiple spaces\n",
			expected: "=",
		},
		{
			name:     "mixed delimiters in content",
			content:  "text with > and = and * symbols\n",
			expected: ">",
		},
		{
			name:     "delimiter-like but not at start",
			content:  "text > not at start\nmore = text\n",
			expected: ">",
		},
		{
			name:     "very long line starting with delimiter",
			content:  "> " + strings.Repeat("a", 10000) + "\n",
			expected: "=",
		},
		{
			name:     "unicode content",
			content:  "unicode: 中文 🚀 ñoño\n",
			expected: ">",
		},
		{
			name:     "all single length delimiters conflict",
			content:  "> conflicts\n= conflicts\n* conflicts\n- conflicts\n",
			expected: ">>",
		},
		{
			name:     "prefers shorter delimiter from different char",
			content:  "> c\n>> c\n>>> c\n>>>> c\n>>>>> c\n",
			expected: "=",
		},
		{
			name:     "prefer = over >> when > conflicts",
			content:  "> conflicts but = is free\n",
			expected: "=",
		},
		{
			name:     "prefer * over == when > and = conflict",
			content:  "> conflicts\n= also conflicts\n",
			expected: "*",
		},
		{
			name:     "prefer - when >=* conflict",
			content:  "> conflicts\n= conflicts\n* conflicts\n",
			expected: "-",
		},
		{
			name:     "fallback to >> when all single chars conflict",
			content:  "> conflicts\n= conflicts\n* conflicts\n- conflicts\n",
			expected: ">>",
		},
		{
			name:     "complex interleaving",
			content:  "> a\n== b\n*** c\n---- d\n>>>>> e\n",
			expected: "=",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := &TortiseDocument{
				Files: []TortiseFile{
					{Path: "test.txt", Content: test.content},
				},
			}
			
			result, err := findSafeDelimiter(doc)
			if err != nil {
				t.Fatalf("findSafeDelimiter failed: %v", err)
			}
			
			if result != test.expected {
				t.Errorf("Expected delimiter %q, got %q", test.expected, result)
			}
		})
	}
}

func TestAutoDiscoveryMultipleFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []TortiseFile
		expected string
	}{
		{
			name: "conflicts across multiple files",
			files: []TortiseFile{
				{Path: "file1.txt", Content: "> conflict in file 1\n"},
				{Path: "file2.txt", Content: "= conflict in file 2\n"},
			},
			expected: "*",
		},
		{
			name: "one file empty, one with conflicts",
			files: []TortiseFile{
				{Path: "empty.txt", Content: ""},
				{Path: "conflict.txt", Content: "> has conflict\n"},
			},
			expected: "=",
		},
		{
			name: "many files, deep conflicts",
			files: []TortiseFile{
				{Path: "f1.txt", Content: "> c\n>> c\n>>> c\n>>>> c\n"},
				{Path: "f2.txt", Content: "= c\n== c\n=== c\n==== c\n"},
				{Path: "f3.txt", Content: "* c\n** c\n*** c\n"},
				{Path: "f4.txt", Content: "- c\n-- c\n"},
			},
			expected: "---",
		},
		{
			name: "scattered conflicts",
			files: []TortiseFile{
				{Path: "f1.txt", Content: "normal content\n"},
				{Path: "f2.txt", Content: "> conflict here\nother content\n"},
				{Path: "f3.txt", Content: "more normal\n= another conflict\n"},
			},
			expected: "*",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := &TortiseDocument{Files: test.files}
			
			result, err := findSafeDelimiter(doc)
			if err != nil {
				t.Fatalf("findSafeDelimiter failed: %v", err)
			}
			
			if result != test.expected {
				t.Errorf("Expected delimiter %q, got %q", test.expected, result)
			}
		})
	}
}

func TestAutoDiscoveryExtremeCases(t *testing.T) {
	t.Run("conflict at maximum length", func(t *testing.T) {
		content := strings.Repeat(">", 50) + " conflict at max length\n"
		
		doc := &TortiseDocument{
			Files: []TortiseFile{
				{Path: "test.txt", Content: content},
			},
		}
		
		result, err := findSafeDelimiter(doc)
		if err != nil {
			t.Fatalf("findSafeDelimiter failed: %v", err)
		}
		
		if result != ">" {
			t.Errorf("Expected '>' when only max-length > conflicts, got %q", result)
		}
	})
	
	t.Run("conflicts up to length 49", func(t *testing.T) {
		content := ""
		for i := 1; i < 50; i++ {
			content += strings.Repeat(">", i) + " conflict\n"
		}
		
		doc := &TortiseDocument{
			Files: []TortiseFile{
				{Path: "test.txt", Content: content},
			},
		}
		
		result, err := findSafeDelimiter(doc)
		if err != nil {
			t.Fatalf("findSafeDelimiter failed: %v", err)
		}
		
		if result != "=" {
			t.Errorf("Expected '=' when all > lengths 1-49 conflict, got %q", result)
		}
	})
	
	t.Run("systematic elimination", func(t *testing.T) {
		// Eliminate all > up to length 10, all = up to 5, all * up to 3
		content := ""
		for i := 1; i <= 10; i++ {
			content += strings.Repeat(">", i) + " conflict\n"
		}
		for i := 1; i <= 5; i++ {
			content += strings.Repeat("=", i) + " conflict\n"
		}
		for i := 1; i <= 3; i++ {
			content += strings.Repeat("*", i) + " conflict\n"
		}
		
		doc := &TortiseDocument{
			Files: []TortiseFile{
				{Path: "test.txt", Content: content},
			},
		}
		
		result, err := findSafeDelimiter(doc)
		if err != nil {
			t.Fatalf("findSafeDelimiter failed: %v", err)
		}
		
		if result != "-" {
			t.Errorf("Expected '-' after systematic elimination, got %q", result)
		}
	})
}

func TestAutoDiscoveryIntegrationWithWriteTo(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		shouldContain  string
		shouldNotStart string
	}{
		{
			name:           "simple conflict resolution",
			content:        "> this conflicts\nnormal content\n",
			shouldContain:  "= test.txt\n",
			shouldNotStart: "> test.txt\n",
		},
		{
			name:           "multiple conflicts resolved",
			content:        "> conflicts\n= also conflicts\nnormal\n",
			shouldContain:  "* test.txt\n",
			shouldNotStart: "> test.txt\n",
		},
		{
			name:           "no conflicts uses default",
			content:        "normal content\nno conflicts here\n",
			shouldContain:  "> test.txt\n",
			shouldNotStart: "= test.txt\n",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := &TortiseDocument{
				Files: []TortiseFile{
					{Path: "test.txt", Content: test.content},
				},
			}
			
			var buf strings.Builder
			err := doc.WriteTo(&buf)
			if err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			
			output := buf.String()
			
			if !strings.Contains(output, test.shouldContain) {
				t.Errorf("Output should contain %q, got:\n%s", test.shouldContain, output)
			}
			
			if strings.HasPrefix(output, test.shouldNotStart) {
				t.Errorf("Output should not start with %q, got:\n%s", test.shouldNotStart, output[:50])
			}
		})
	}
}

func TestManualDelimiterOverrideVsAutoDiscovery(t *testing.T) {
	content := "> this would conflict with auto-discovery\n"
	
	t.Run("auto discovery avoids conflict", func(t *testing.T) {
		doc := &TortiseDocument{
			Files: []TortiseFile{
				{Path: "test.txt", Content: content},
			},
		}
		
		var buf strings.Builder
		err := doc.WriteTo(&buf)
		if err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		
		if strings.HasPrefix(buf.String(), "> test.txt\n") {
			t.Error("Auto-discovery should have avoided > delimiter")
		}
	})
	
	t.Run("manual override causes collision error", func(t *testing.T) {
		doc := &TortiseDocument{
			Delimiter: ">",
			Files: []TortiseFile{
				{Path: "test.txt", Content: content},
			},
		}
		
		var buf strings.Builder
		err := doc.WriteTo(&buf)
		if err == nil {
			t.Error("Expected collision error with manual delimiter")
		}
		
		if !strings.Contains(err.Error(), "conflicts with content") {
			t.Errorf("Expected collision error, got: %v", err)
		}
	})
}

func TestDelimiterPreferenceOrder(t *testing.T) {
	// Test that at the same length, preference is >, =, *, -
	chars := []rune{'>', '=', '*', '-'}
	
	for i := 0; i < len(chars); i++ {
		t.Run(fmt.Sprintf("prefer_%c_over_later_chars", chars[i]), func(t *testing.T) {
			content := ""
			// Block all characters before the target
			for j := 0; j < i; j++ {
				content += string(chars[j]) + " blocked\n"
			}
			
			doc := &TortiseDocument{
				Files: []TortiseFile{
					{Path: "test.txt", Content: content},
				},
			}
			
			result, err := findSafeDelimiter(doc)
			if err != nil {
				t.Fatalf("findSafeDelimiter failed: %v", err)
			}
			
			expected := string(chars[i])
			if result != expected {
				t.Errorf("Expected %q (first available), got %q", expected, result)
			}
		})
	}
}

func TestPerformanceWithLargeContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Create a large file with many lines but no conflicts
	lines := make([]string, 10000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d with normal content", i)
	}
	content := strings.Join(lines, "\n") + "\n"
	
	doc := &TortiseDocument{
		Files: []TortiseFile{
			{Path: "large.txt", Content: content},
		},
	}
	
	start := time.Now()
	result, err := findSafeDelimiter(doc)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("findSafeDelimiter failed: %v", err)
	}
	
	if result != ">" {
		t.Errorf("Expected '>' for content with no conflicts, got %q", result)
	}
	
	if elapsed > 100*time.Millisecond {
		t.Errorf("Auto-discovery took too long: %v", elapsed)
	}
}

func TestImprovedErrorMessages(t *testing.T) {
	t.Run("helpful error with auto-suggestion", func(t *testing.T) {
		doc := &TortiseDocument{
			Delimiter: ">",
			Files: []TortiseFile{
				{Path: "conflict.txt", Content: "> this conflicts\nnormal content\n"},
			},
		}
		
		var buf strings.Builder
		err := doc.WriteTo(&buf)
		if err == nil {
			t.Error("Expected collision error")
		}
		
		errMsg := err.Error()
		expectedParts := []string{
			"delimiter \">\" conflicts with content",
			"conflict.txt",
			"auto-generated delimiter \"=\"",
			"remove -d flag",
			"choose a different delimiter",
		}
		
		for _, part := range expectedParts {
			if !strings.Contains(errMsg, part) {
				t.Errorf("Error message missing %q. Got: %s", part, errMsg)
			}
		}
	})
	
	t.Run("error when auto-generation impossible", func(t *testing.T) {
		// Create content that conflicts with ALL possible delimiters
		content := ""
		for _, char := range []rune{'>', '=', '*', '-'} {
			for length := 1; length <= 50; length++ {
				delimiter := strings.Repeat(string(char), length)
				content += delimiter + " conflicts\n"
			}
		}
		
		doc := &TortiseDocument{
			Delimiter: ">",
			Files: []TortiseFile{
				{Path: "impossible.txt", Content: content},
			},
		}
		
		var buf strings.Builder
		err := doc.WriteTo(&buf)
		if err == nil {
			t.Error("Expected collision error")
		}
		
		errMsg := err.Error()
		expectedParts := []string{
			"delimiter \">\" conflicts with content",
			"impossible.txt",
			"no safe delimiter could be auto-generated",
			"all delimiters up to 50 characters conflict",
		}
		
		for _, part := range expectedParts {
			if !strings.Contains(errMsg, part) {
				t.Errorf("Error message missing %q. Got: %s", part, errMsg)
			}
		}
	})
}
