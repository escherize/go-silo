package tortise_go

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
