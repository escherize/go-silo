package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/escherize/silo"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "reap":
		reapCmd()
	case "sow":
		sowCmd()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func reapCmd() {
	reapFlags := flag.NewFlagSet("reap", flag.ExitOnError)
	outputFile := reapFlags.String("o", "", "Output silo file (default: stdout)")
	delimiter := reapFlags.String("d", "", "Delimiter to use (auto-detected if not specified)")
	useEnhanced := reapFlags.Bool("enhanced", false, "Use enhanced glob support with ** patterns")
	
	reapFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: silo reap [options] <pattern1 pattern2 ...>\n")
		fmt.Fprintf(os.Stderr, "Reap files matching glob patterns into a silo file\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		reapFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  silo reap src/                          Reap directory\n")
		fmt.Fprintf(os.Stderr, "  silo reap \"*.go\" \"*.md\"                   Reap multiple patterns\n")
		fmt.Fprintf(os.Stderr, "  silo reap -enhanced \"src/**/*.go\"         Reap with recursive ** pattern\n")
		fmt.Fprintf(os.Stderr, "  silo reap -d \"🌾\" \"*.txt\" -o out.silo     Reap with wheat emoji delimiter\n")
		fmt.Fprintf(os.Stderr, "  silo reap \"a/this\" \"b/that\"              Reap specific paths\n")
		fmt.Fprintf(os.Stderr, "\nSecurity: Patterns with .. or absolute paths are rejected\n")
	}
	
	reapFlags.Parse(os.Args[2:])
	
	if reapFlags.NArg() < 1 {
		reapFlags.Usage()
		os.Exit(1)
	}
	
	// Create secure glob expander
	globber, err := silo.NewSecureGlobExpander()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing glob expander: %v\n", err)
		os.Exit(1)
	}
	
	// Collect all patterns
	patterns := make([]string, reapFlags.NArg())
	for i := 0; i < reapFlags.NArg(); i++ {
		patterns[i] = reapFlags.Arg(i)
	}
	
	// Choose glob option based on flags
	var globOption silo.GlobOption
	if *useEnhanced {
		globOption = silo.EnhancedGlob
	} else {
		globOption = silo.BothGlobs // Try enhanced, fall back to standard
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
	var doc *silo.SiloDocument
	if len(filePaths) == 1 {
		if info, statErr := os.Stat(filePaths[0]); statErr == nil && info.IsDir() {
			doc, err = silo.ReadDirectoryTree(filePaths[0])
		} else {
			doc, err = silo.ReadFiles(filePaths)
		}
	} else {
		// Multiple files/patterns
		doc, err = silo.ReadFiles(filePaths)
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
		fmt.Fprintf(os.Stderr, "Error writing silo file: %v\n", err)
		os.Exit(1)
	}
}

func sowCmd() {
	sowFlags := flag.NewFlagSet("sow", flag.ExitOnError)
	outputDir := sowFlags.String("o", ".", "Output directory")
	
	sowFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: silo sow [options] <silo-file>\n")
		fmt.Fprintf(os.Stderr, "Sow a silo file into a directory tree\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		sowFlags.PrintDefaults()
	}
	
	sowFlags.Parse(os.Args[2:])
	
	if sowFlags.NArg() != 1 {
		sowFlags.Usage()
		os.Exit(1)
	}
	
	siloFile := sowFlags.Arg(0)
	
	file, err := os.Open(siloFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening silo file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	
	doc, err := silo.ParseSiloFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing silo file: %v\n", err)
		os.Exit(1)
	}
	
	if err := doc.WriteToDirectory(*outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to directory: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully unpacked %d files to %s\n", len(doc.Files), *outputDir)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "silo - A tool for reaping/sowing directory trees and files\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  silo reap [options] <pattern1 pattern2 ...>    Reap files into silo file\n")
	fmt.Fprintf(os.Stderr, "  silo sow [options] <file>                      Sow silo file into directory\n")
	fmt.Fprintf(os.Stderr, "  silo help                                      Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  silo reap src/ -o project.silo                 Reap 'src' directory (auto-detect delimiter)\n")
	fmt.Fprintf(os.Stderr, "  silo reap \"*.go\" \"*.md\"                        Reap multiple patterns with auto-detected delimiter\n")
	fmt.Fprintf(os.Stderr, "  silo reap -d \"🌾\" \"*.go\" -o code.silo          Reap with wheat emoji delimiter\n")
	fmt.Fprintf(os.Stderr, "  silo sow project.silo                          Sow to current directory\n")
	fmt.Fprintf(os.Stderr, "  silo sow project.silo -o out/                  Sow to 'out' directory\n")
}
