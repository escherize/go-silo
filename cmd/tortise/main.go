package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/escherize/tortise_go"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "pack":
		packCmd()
	case "unpack":
		unpackCmd()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func packCmd() {
	packFlags := flag.NewFlagSet("pack", flag.ExitOnError)
	outputFile := packFlags.String("o", "", "Output tortise file (default: stdout)")
	delimiter := packFlags.String("d", "", "Delimiter to use (auto-detected if not specified)")
	useEnhanced := packFlags.Bool("enhanced", false, "Use enhanced glob support with ** patterns")
	
	packFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tortise pack [options] <pattern1 pattern2 ...>\n")
		fmt.Fprintf(os.Stderr, "Pack files matching glob patterns into a tortise file\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		packFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  tortise pack src/                          Pack directory\n")
		fmt.Fprintf(os.Stderr, "  tortise pack \"*.go\" \"*.md\"                 Pack multiple patterns\n")
		fmt.Fprintf(os.Stderr, "  tortise pack -enhanced \"src/**/*.go\"        Pack with recursive ** pattern\n")
		fmt.Fprintf(os.Stderr, "  tortise pack -d \"🐢\" \"*.txt\" -o out.tortise  Pack with emoji delimiter\n")
		fmt.Fprintf(os.Stderr, "  tortise pack \"a/this\" \"b/that\"              Pack specific paths\n")
		fmt.Fprintf(os.Stderr, "\nSecurity: Patterns with .. or absolute paths are rejected\n")
	}
	
	packFlags.Parse(os.Args[2:])
	
	if packFlags.NArg() < 1 {
		packFlags.Usage()
		os.Exit(1)
	}
	
	// Create secure glob expander
	globber, err := tortise_go.NewSecureGlobExpander()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing glob expander: %v\n", err)
		os.Exit(1)
	}
	
	// Collect all patterns
	patterns := make([]string, packFlags.NArg())
	for i := 0; i < packFlags.NArg(); i++ {
		patterns[i] = packFlags.Arg(i)
	}
	
	// Choose glob option based on flags
	var globOption tortise_go.GlobOption
	if *useEnhanced {
		globOption = tortise_go.EnhancedGlob
	} else {
		globOption = tortise_go.BothGlobs // Try enhanced, fall back to standard
	}
	
	// Expand patterns safely
	filePaths, err := globber.ExpandPatterns(patterns, globOption)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expanding patterns: %v\n", err)
		os.Exit(1)
	}
	
	if len(filePaths) == 0 {
		fmt.Fprintf(os.Stderr, "No files matched the specified patterns\n")
		os.Exit(1)
	}
	
	// Check if we have a single directory
	var doc *tortise_go.TortiseDocument
	if len(filePaths) == 1 {
		if info, statErr := os.Stat(filePaths[0]); statErr == nil && info.IsDir() {
			doc, err = tortise_go.ReadDirectoryTree(filePaths[0])
		} else {
			doc, err = tortise_go.ReadFiles(filePaths)
		}
	} else {
		// Multiple files/patterns
		doc, err = tortise_go.ReadFiles(filePaths)
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
	
	if *delimiter != "" {
		doc.Delimiter = *delimiter
	} else {
		doc.Delimiter = ""
	}
	
	if *outputFile == "" {
		err = doc.WriteTo(os.Stdout)
	} else {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		
		err = doc.WriteTo(file)
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing tortise file: %v\n", err)
		os.Exit(1)
	}
}

func unpackCmd() {
	unpackFlags := flag.NewFlagSet("unpack", flag.ExitOnError)
	outputDir := unpackFlags.String("o", ".", "Output directory")
	
	unpackFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tortise unpack [options] <tortise-file>\n")
		fmt.Fprintf(os.Stderr, "Unpack a tortise file into a directory tree\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		unpackFlags.PrintDefaults()
	}
	
	unpackFlags.Parse(os.Args[2:])
	
	if unpackFlags.NArg() != 1 {
		unpackFlags.Usage()
		os.Exit(1)
	}
	
	tortiseFile := unpackFlags.Arg(0)
	
	file, err := os.Open(tortiseFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening tortise file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	
	doc, err := tortise_go.ParseTortiseFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing tortise file: %v\n", err)
		os.Exit(1)
	}
	
	if err := doc.WriteToDirectory(*outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to directory: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully unpacked %d files to %s\n", len(doc.Files), *outputDir)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "tortise - A tool for packing/unpacking directory trees and files\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  tortise pack [options] <directory|file1 file2 ...>  Pack directory or files into tortise file\n")
	fmt.Fprintf(os.Stderr, "  tortise unpack [options] <file>                     Unpack tortise file into directory\n")
	fmt.Fprintf(os.Stderr, "  tortise help                                        Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  tortise pack src/ -o project.tortise               Pack 'src' directory (auto-detect delimiter)\n")
	fmt.Fprintf(os.Stderr, "  tortise pack *.go                                   Pack all .go files with auto-detected delimiter\n")
	fmt.Fprintf(os.Stderr, "  tortise pack -d \">>>\" *.go -o code.tortise          Pack with specific delimiter\n")
	fmt.Fprintf(os.Stderr, "  tortise unpack project.tortise                      Unpack to current directory\n")
	fmt.Fprintf(os.Stderr, "  tortise unpack project.tortise -o out/              Unpack to 'out' directory\n")
}