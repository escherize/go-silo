package silo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecureGlobExpanderValidation(t *testing.T) {
	expander, err := NewSecureGlobExpander()
	if err != nil {
		t.Fatalf("Failed to create SecureGlobExpander: %v", err)
	}

	tests := []struct {
		name        string
		pattern     string
		shouldError bool
		errorText   string
	}{
		// Valid patterns
		{"simple glob", "*.go", false, ""},
		{"directory", "src/", false, ""},
		{"subdirectory glob", "src/*.go", false, ""},
		{"unicode characters", "测试/*.txt", false, ""},
		{"emoji patterns", "🐢*.go", false, ""},
		
		// Invalid patterns - absolute paths
		{"absolute path unix", "/etc/passwd", true, "absolute paths not allowed"},
		{"absolute path windows", "C:\\Windows\\System32", true, "drive letters not allowed"},
		{"absolute path with glob", "/home/*", true, "absolute paths not allowed"},
		
		// Invalid patterns - parent directory references  
		{"parent reference", "../config", true, "parent directory references not allowed"},
		{"parent with glob", "../*.txt", true, "parent directory references not allowed"},
		{"deep parent reference", "../../etc/passwd", true, "parent directory references not allowed"},
		{"parent in middle", "src/../config", true, "parent directory references not allowed"},
		{"unix path traversal", "src/../../../etc/passwd", true, "parent directory references not allowed"},
		{"windows path traversal", "src\\..\\..\\windows", true, "parent directory references not allowed"},
		
		// Edge cases
		{"just dots", "..", true, "parent directory references not allowed"},
		{"dots with slash", "../", true, "parent directory references not allowed"},
		{"relative current", "./file.txt", false, ""},
		{"hidden files", ".gitignore", false, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expander.ValidatePattern(test.pattern)
			
			if test.shouldError {
				if err == nil {
					t.Errorf("Expected error for pattern %q, but got none", test.pattern)
				} else if !strings.Contains(err.Error(), test.errorText) {
					t.Errorf("Expected error to contain %q, got: %v", test.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for pattern %q, but got: %v", test.pattern, err)
				}
			}
		})
	}
}

func TestSecureGlobExpanderPathValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create expander with temp directory as working directory
	expander := &SecureGlobExpander{
		AllowAbsolute: false,
		WorkingDir:    tempDir,
	}
	
	// Create some test files within tempDir
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("sub"), 0644); err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		shouldError bool
	}{
		// Valid paths (within working directory)
		{"file in working dir", "test.txt", false},
		{"absolute path to file in working dir", testFile, false},
		{"subdirectory file", filepath.Join("subdir", "sub.txt"), false},
		{"absolute path to subdirectory file", subFile, false},
		{"subdirectory", "subdir", false},
		
		// Invalid paths (outside working directory)
		{"parent directory", "..", true},
		{"file in parent", "../file.txt", true},
		{"absolute path outside", "/etc/passwd", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expander.ValidatePath(test.path)
			
			if test.shouldError {
				if err == nil {
					t.Errorf("Expected error for path %q, but got none", test.path)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for path %q, but got: %v", test.path, err)
				}
			}
		})
	}
}

func TestExpandPatternsIntegration(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	
	// Change to temp directory for relative path testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// Create test file structure
	files := map[string]string{
		"file1.go":           "package main",
		"file2.go":           "package main",
		"README.md":          "# Test",
		"src/main.go":        "package main",
		"src/util.go":        "package main", 
		"test/unit_test.go":  "package main",
		"docs/guide.md":      "# Guide",
		"scripts/build.sh":   "#!/bin/bash",
	}
	
	for path, content := range files {
		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}
	
	expander, err := NewSecureGlobExpander()
	if err != nil {
		t.Fatalf("Failed to create expander: %v", err)
	}

	tests := []struct {
		name            string
		patterns        []string
		option          GlobOption
		expectedCount   int
		expectedFiles   []string
		shouldError     bool
	}{
		{
			name:          "single glob pattern",
			patterns:      []string{"*.go"},
			option:        StandardGlob,
			expectedCount: 2,
			expectedFiles: []string{"file1.go", "file2.go"},
		},
		{
			name:          "multiple patterns",
			patterns:      []string{"*.go", "*.md"},
			option:        StandardGlob,
			expectedCount: 3,
			expectedFiles: []string{"file1.go", "file2.go", "README.md"},
		},
		{
			name:          "subdirectory pattern",
			patterns:      []string{"src/*.go"},
			option:        StandardGlob,
			expectedCount: 2,
			expectedFiles: []string{"src/main.go", "src/util.go"},
		},
		{
			name:          "enhanced recursive pattern",
			patterns:      []string{"**/*.go"},
			option:        EnhancedGlob,
			expectedCount: 4, // file1.go, file2.go, src/main.go, src/util.go, test/unit_test.go
		},
		{
			name:        "invalid parent reference",
			patterns:    []string{"../*.txt"},
			option:      StandardGlob,
			shouldError: true,
		},
		{
			name:        "absolute path pattern",
			patterns:    []string{"/etc/*"},
			option:      StandardGlob,
			shouldError: true,
		},
		{
			name:          "literal file names",
			patterns:      []string{"README.md", "src/main.go"},
			option:        StandardGlob,
			expectedCount: 2,
			expectedFiles: []string{"README.md", "src/main.go"},
		},
		{
			name:          "mixed patterns and literals",
			patterns:      []string{"*.md", "src/main.go"},
			option:        StandardGlob,
			expectedCount: 3,
			expectedFiles: []string{"README.md", "docs/guide.md", "src/main.go"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := expander.ExpandPatterns(test.patterns, test.option)
			
			if test.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if len(result) != test.expectedCount {
				t.Errorf("Expected %d files, got %d: %v", test.expectedCount, len(result), result)
			}
			
			// Check that expected files are present (if specified)
			if test.expectedFiles != nil {
				resultSet := make(map[string]bool)
				for _, file := range result {
					resultSet[file] = true
				}
				
				for _, expected := range test.expectedFiles {
					if !resultSet[expected] {
						t.Errorf("Expected file %q not found in results: %v", expected, result)
					}
				}
			}
		})
	}
}

func TestExpandPatternsDeduplication(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// Create a test file
	if err := os.WriteFile("test.go", []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	expander, err := NewSecureGlobExpander()
	if err != nil {
		t.Fatalf("Failed to create expander: %v", err)
	}
	
	// Use patterns that should match the same file
	patterns := []string{"test.go", "*.go", "test.go"}
	result, err := expander.ExpandPatterns(patterns, StandardGlob)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should only have one file despite multiple matching patterns
	if len(result) != 1 {
		t.Errorf("Expected 1 deduplicated file, got %d: %v", len(result), result)
	}
	
	if result[0] != "test.go" {
		t.Errorf("Expected test.go, got %s", result[0])
	}
}

func TestSecurityEscapeAttempts(t *testing.T) {
	expander, err := NewSecureGlobExpander()
	if err != nil {
		t.Fatalf("Failed to create expander: %v", err)
	}
	
	// Various escape attempt patterns that should all be rejected
	maliciousPatterns := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"/etc/passwd",
		"C:\\Windows\\System32\\*",
		"src/../../../etc/passwd",
		"./../../etc/passwd",
		"src\\..\\..\\..\\windows",
		"~/../../../etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2f",  // URL encoded ../../../
	}
	
	for _, pattern := range maliciousPatterns {
		t.Run("malicious_pattern_"+pattern, func(t *testing.T) {
			_, err := expander.ExpandPatterns([]string{pattern}, BothGlobs)
			if err == nil {
				t.Errorf("Expected security error for malicious pattern %q, but expansion succeeded", pattern)
			}
		})
	}
}