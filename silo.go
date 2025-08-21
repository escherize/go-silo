package silo

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type SiloFile struct {
	Path    string
	Content string
}

type SiloDocument struct {
	Files     []SiloFile
	Delimiter string
}

func detectDelimiter(line string) (string, string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", fmt.Errorf("empty line cannot contain delimiter")
	}

	delim := ""
	byteIdx := 0
	
	// Process the line rune by rune to handle Unicode properly
	for byteIdx < len(line) {
		r, size := utf8.DecodeRuneInString(line[byteIdx:])
		if r == utf8.RuneError {
			return "", "", fmt.Errorf("invalid UTF-8 encoding")
		}
		
		if !isValidDelimiterChar(r) {
			break
		}
		
		delim += string(r)
		byteIdx += size
	}
	
	if delim == "" {
		return "", "", fmt.Errorf("invalid file declaration format")
	}
	
	// Check that we have a space after the delimiter
	if byteIdx >= len(line) || line[byteIdx] != ' ' {
		return "", "", fmt.Errorf("invalid file declaration format")
	}
	
	path := strings.TrimSpace(line[byteIdx+1:])
	if path == "" {
		return "", "", fmt.Errorf("empty path")
	}
	
	return delim, path, nil
}

// isValidDelimiterChar returns true if the rune can be part of a delimiter.
// Per spec v0.2: any Unicode character except ASCII space (0x20), horizontal tab (0x09),
// line feed (0x0A), or carriage return (0x0D).
func isValidDelimiterChar(r rune) bool {
	return r != 0x20 && r != 0x09 && r != 0x0A && r != 0x0D
}

func validatePath(path string) error {
	if path == "" || path == "." {
		return fmt.Errorf("invalid path: %s", path)
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("parent directory references not allowed: %s", path)
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("null character in path: %s", path)
	}
	return nil
}

func ParseSiloFile(r io.Reader) (*SiloDocument, error) {
	scanner := bufio.NewScanner(r)
	lines := []string{}
	
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.ReplaceAll(line, "\r\n", "\n")
		line = strings.ReplaceAll(line, "\r", "\n")
		lines = append(lines, line)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	doc := &SiloDocument{}
	pathsSeen := make(map[string]bool)
	
	lineIdx := 0
	for lineIdx < len(lines) && isBlankLine(lines[lineIdx]) {
		lineIdx++
	}
	
	if lineIdx >= len(lines) {
		return doc, nil
	}

	delim, firstPath, err := detectDelimiter(lines[lineIdx])
	if err != nil {
		return nil, fmt.Errorf("error detecting delimiter on line %d: %w", lineIdx+1, err)
	}
	
	doc.Delimiter = delim
	lineIdx++

	if err := validatePath(firstPath); err != nil {
		return nil, fmt.Errorf("invalid path on line %d: %w", lineIdx, err)
	}
	
	if pathsSeen[firstPath] {
		return nil, fmt.Errorf("duplicate path: %s", firstPath)
	}
	pathsSeen[firstPath] = true

	currentFile := &SiloFile{Path: firstPath}
	contentLines := []string{}
	
	for lineIdx < len(lines) {
		line := lines[lineIdx]
		
		if strings.HasPrefix(line, delim+" ") {
			currentFile.Content = strings.Join(contentLines, "\n")
			if currentFile.Content != "" {
				currentFile.Content += "\n"
			}
			doc.Files = append(doc.Files, *currentFile)
			
			path := strings.TrimSpace(line[len(delim)+1:])
			if err := validatePath(path); err != nil {
				return nil, fmt.Errorf("invalid path on line %d: %w", lineIdx+1, err)
			}
			
			if pathsSeen[path] {
				return nil, fmt.Errorf("duplicate path: %s", path)
			}
			pathsSeen[path] = true
			
			currentFile = &SiloFile{Path: path}
			contentLines = []string{}
		} else {
			contentLines = append(contentLines, line)
		}
		lineIdx++
	}
	
	currentFile.Content = strings.Join(contentLines, "\n")
	if currentFile.Content != "" {
		currentFile.Content += "\n"
	}
	doc.Files = append(doc.Files, *currentFile)
	
	return doc, nil
}

func findSafeDelimiter(doc *SiloDocument) (string, error) {
	baseChars := []rune{'>', '=', '*', '-'}
	candidates := make(map[string]bool)
	
	for _, char := range baseChars {
		for length := 1; length <= 50; length++ {
			delimiter := strings.Repeat(string(char), length)
			candidates[delimiter] = true
		}
	}
	
	for _, file := range doc.Files {
		for _, line := range strings.Split(file.Content, "\n") {
			if line == "" {
				continue
			}
			
			for delimiter := range candidates {
				if strings.HasPrefix(line, delimiter+" ") {
					delete(candidates, delimiter)
				}
			}
		}
	}
	
	if len(candidates) == 0 {
		return "", fmt.Errorf("unable to find safe delimiter: all delimiters up to 50 characters conflict with file content")
	}
	
	shortestLength := 51
	for delimiter := range candidates {
		if len(delimiter) < shortestLength {
			shortestLength = len(delimiter)
		}
	}
	
	preferences := []rune{'>', '=', '*', '-'}
	for _, char := range preferences {
		delimiter := strings.Repeat(string(char), shortestLength)
		if candidates[delimiter] {
			return delimiter, nil
		}
	}
	
	for delimiter := range candidates {
		if len(delimiter) == shortestLength {
			return delimiter, nil
		}
	}
	
	return "", fmt.Errorf("internal error: no delimiter found despite having candidates")
}

func (doc *SiloDocument) WriteTo(w io.Writer) error {
	wasAutoDetected := doc.Delimiter == ""
	if doc.Delimiter == "" {
		delimiter, err := findSafeDelimiter(doc)
		if err != nil {
			return err
		}
		doc.Delimiter = delimiter
	}
	
	if !wasAutoDetected {
		for _, file := range doc.Files {
			for _, line := range strings.Split(file.Content, "\n") {
				if line != "" && strings.HasPrefix(line, doc.Delimiter+" ") {
					autoDelimiter, autoErr := findSafeDelimiter(doc)
					if autoErr != nil {
						return fmt.Errorf("delimiter %q conflicts with content in file %s, and no safe delimiter could be auto-generated: %v", doc.Delimiter, file.Path, autoErr)
					}
					return fmt.Errorf("delimiter %q conflicts with content in file %s. Try using auto-generated delimiter %q (remove -d flag) or choose a different delimiter", doc.Delimiter, file.Path, autoDelimiter)
				}
			}
		}
	}
	
	for _, file := range doc.Files {
		_, err := fmt.Fprintf(w, "%s %s\n", doc.Delimiter, file.Path)
		if err != nil {
			return err
		}
		
		content := file.Content
		if !strings.HasSuffix(content, "\n") && content != "" {
			content += "\n"
		}
		
		_, err = w.Write([]byte(content))
		if err != nil {
			return err
		}
	}
	
	return nil
}

func isBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

func ReadDirectoryTree(rootPath string) (*SiloDocument, error) {
	doc := &SiloDocument{Delimiter: ">"}
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		
		relPath = filepath.ToSlash(relPath)
		
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		doc.Files = append(doc.Files, SiloFile{
			Path:    relPath,
			Content: string(content),
		})
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	sort.Slice(doc.Files, func(i, j int) bool {
		return doc.Files[i].Path < doc.Files[j].Path
	})
	
	return doc, nil
}

func ReadFiles(filePaths []string) (*SiloDocument, error) {
	doc := &SiloDocument{Delimiter: ">"}
	
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}
		
		if info.IsDir() {
			return nil, fmt.Errorf("path %s is a directory, not a file", filePath)
		}
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		
		doc.Files = append(doc.Files, SiloFile{
			Path:    filepath.ToSlash(filePath),
			Content: string(content),
		})
	}
	
	sort.Slice(doc.Files, func(i, j int) bool {
		return doc.Files[i].Path < doc.Files[j].Path
	})
	
	return doc, nil
}

func (doc *SiloDocument) WriteToDirectory(rootPath string) error {
	for _, file := range doc.Files {
		fullPath := filepath.Join(rootPath, filepath.FromSlash(file.Path))
		
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		if err := os.WriteFile(fullPath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}
	
	return nil
}
