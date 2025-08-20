package tortise_go

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TortiseFile struct {
	Path    string
	Content string
}

type TortiseDocument struct {
	Files     []TortiseFile
	Delimiter string
}

func detectDelimiter(line string) (string, string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", fmt.Errorf("empty line cannot contain delimiter")
	}

	delim := ""
	i := 0
	for i < len(line) && isPunctuation(rune(line[i])) {
		delim += string(line[i])
		i++
	}
	
	if delim == "" {
		return "", "", fmt.Errorf("invalid file declaration format")
	}
	
	if i >= len(line) || line[i] != ' ' {
		return "", "", fmt.Errorf("invalid file declaration format")
	}
	
	path := strings.TrimSpace(line[i+1:])
	if path == "" {
		return "", "", fmt.Errorf("empty path")
	}
	
	return delim, path, nil
}

func isPunctuation(r rune) bool {
	if r > 127 {
		return false
	}
	return (r >= 33 && r <= 47) || (r >= 58 && r <= 64) || (r >= 91 && r <= 96) || (r >= 123 && r <= 126)
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

func ParseTortiseFile(r io.Reader) (*TortiseDocument, error) {
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

	doc := &TortiseDocument{}
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

	currentFile := &TortiseFile{Path: firstPath}
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
			
			currentFile = &TortiseFile{Path: path}
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

func findSafeDelimiter(doc *TortiseDocument) (string, error) {
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

func (doc *TortiseDocument) WriteTo(w io.Writer) error {
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

func ReadDirectoryTree(rootPath string) (*TortiseDocument, error) {
	doc := &TortiseDocument{Delimiter: ">"}
	
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
		
		doc.Files = append(doc.Files, TortiseFile{
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

func ReadFiles(filePaths []string) (*TortiseDocument, error) {
	doc := &TortiseDocument{Delimiter: ">"}
	
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
		
		doc.Files = append(doc.Files, TortiseFile{
			Path:    filepath.ToSlash(filePath),
			Content: string(content),
		})
	}
	
	sort.Slice(doc.Files, func(i, j int) bool {
		return doc.Files[i].Path < doc.Files[j].Path
	})
	
	return doc, nil
}

func (doc *TortiseDocument) WriteToDirectory(rootPath string) error {
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
