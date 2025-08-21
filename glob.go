package silo

import (
	"fmt"
	"path/filepath"
	"strings"
	"os"
	"net/url"
	
	"github.com/bmatcuk/doublestar/v4"
)

// SecureGlobExpander handles glob pattern expansion with security validation
type SecureGlobExpander struct {
	// AllowAbsolute controls whether absolute paths are allowed (default: false)
	AllowAbsolute bool
	// WorkingDir is the base directory for relative path validation
	WorkingDir string
}

// NewSecureGlobExpander creates a new expander with default security settings
func NewSecureGlobExpander() (*SecureGlobExpander, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	
	return &SecureGlobExpander{
		AllowAbsolute: false,
		WorkingDir:    wd,
	}, nil
}

// ValidatePattern checks if a glob pattern is safe according to Silo spec
func (sge *SecureGlobExpander) ValidatePattern(pattern string) error {
	// Check for URL-encoded patterns and decode them
	if strings.Contains(pattern, "%") {
		if decoded, err := url.QueryUnescape(pattern); err == nil {
			// Recursively validate the decoded pattern
			if err := sge.ValidatePattern(decoded); err != nil {
				return fmt.Errorf("URL-encoded pattern contains forbidden content: %s", pattern)
			}
		}
	}
	
	// Check for absolute paths (forbidden by spec)
	if filepath.IsAbs(pattern) && !sge.AllowAbsolute {
		return fmt.Errorf("absolute paths not allowed: %s", pattern)
	}
	
	// Check for parent directory references (forbidden by spec)
	if strings.Contains(pattern, "..") {
		return fmt.Errorf("parent directory references not allowed: %s", pattern)
	}
	
	// Check for leading slash on non-Windows (indicates absolute path)
	if strings.HasPrefix(pattern, "/") && !sge.AllowAbsolute {
		return fmt.Errorf("absolute paths not allowed: %s", pattern)
	}
	
	// Check for Windows drive letters (C:, D:, etc.)
	if len(pattern) >= 2 && pattern[1] == ':' && 
		((pattern[0] >= 'A' && pattern[0] <= 'Z') || (pattern[0] >= 'a' && pattern[0] <= 'z')) {
		return fmt.Errorf("drive letters not allowed: %s", pattern)
	}
	
	// Additional checks for dangerous patterns
	if strings.Contains(pattern, "\\..\\") || strings.Contains(pattern, "/../") {
		return fmt.Errorf("path traversal attempt detected: %s", pattern)
	}
	
	return nil
}

// ValidatePath checks if a resolved path is safe according to Silo spec
func (sge *SecureGlobExpander) ValidatePath(path string) error {
	// First check the pattern itself for obvious violations
	if err := sge.ValidatePattern(path); err != nil {
		return err
	}
	
	// For relative paths, we need to be more permissive during expansion
	// The main goal is to prevent escaping the working directory tree
	if !filepath.IsAbs(path) {
		// Check if relative path contains unsafe components
		parts := strings.Split(filepath.ToSlash(path), "/")
		for _, part := range parts {
			if part == ".." {
				return fmt.Errorf("path %s contains parent directory reference", path)
			}
		}
		return nil
	}
	
	// For absolute paths, ensure they're within the working directory
	absWorkingDir, err := filepath.Abs(sge.WorkingDir)
	if err != nil {
		return fmt.Errorf("failed to resolve working directory: %w", err)
	}
	
	// Check if the absolute path is within the working directory tree
	relPath, err := filepath.Rel(absWorkingDir, path)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}
	
	// If relPath starts with "..", it's outside the working directory
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path %s resolves outside working directory", path)
	}
	
	return nil
}

// GlobOption represents different glob expansion strategies
type GlobOption int

const (
	// StandardGlob uses Go's built-in filepath.Glob
	StandardGlob GlobOption = iota
	// EnhancedGlob uses doublestar for ** support and more features
	EnhancedGlob
	// BothGlobs tries enhanced first, falls back to standard
	BothGlobs
)

// ExpandPatterns expands multiple glob patterns safely
func (sge *SecureGlobExpander) ExpandPatterns(patterns []string, option GlobOption) ([]string, error) {
	var allFiles []string
	seenFiles := make(map[string]bool) // deduplicate results
	
	for _, pattern := range patterns {
		// First validate the pattern itself
		if err := sge.ValidatePattern(pattern); err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		
		var matches []string
		var err error
		
		switch option {
		case StandardGlob:
			matches, err = sge.expandStandardGlob(pattern)
		case EnhancedGlob:
			matches, err = sge.expandEnhancedGlob(pattern)
		case BothGlobs:
			// Try enhanced first, fall back to standard
			matches, err = sge.expandEnhancedGlob(pattern)
			if err != nil {
				matches, err = sge.expandStandardGlob(pattern)
			}
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to expand pattern %q: %w", pattern, err)
		}
		
		// If no matches found, treat as literal path (if it exists)
		if len(matches) == 0 {
			if _, statErr := os.Stat(pattern); statErr == nil {
				matches = []string{pattern}
			}
		}
		
		// Validate all resolved paths
		for _, match := range matches {
			if err := sge.ValidatePath(match); err != nil {
				return nil, fmt.Errorf("unsafe path in results: %w", err)
			}
			
			// Normalize path for consistency - use forward slashes and make relative if possible
			normalizedPath := filepath.ToSlash(match)
			
			// If it's an absolute path within our working directory, make it relative
			if filepath.IsAbs(match) {
				if relPath, err := filepath.Rel(sge.WorkingDir, match); err == nil && !strings.HasPrefix(relPath, "..") {
					normalizedPath = filepath.ToSlash(relPath)
				}
			}
			
			// Deduplicate and add
			if !seenFiles[normalizedPath] {
				seenFiles[normalizedPath] = true
				allFiles = append(allFiles, normalizedPath)
			}
		}
	}
	
	return allFiles, nil
}

// expandStandardGlob uses Go's built-in filepath.Glob
func (sge *SecureGlobExpander) expandStandardGlob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

// expandEnhancedGlob uses doublestar for enhanced glob support
func (sge *SecureGlobExpander) expandEnhancedGlob(pattern string) ([]string, error) {
	// Use doublestar for enhanced glob support with ** and other features
	return doublestar.FilepathGlob(pattern)
}