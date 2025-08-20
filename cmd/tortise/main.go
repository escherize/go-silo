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
	
	packFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tortise pack [options] <directory|file1 file2 ...>\n")
		fmt.Fprintf(os.Stderr, "Pack a directory tree or multiple files into a tortise file\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		packFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  tortise pack src/                      Pack directory (auto-detect delimiter)\n")
		fmt.Fprintf(os.Stderr, "  tortise pack file1.go file2.go         Pack specific files (auto-detect delimiter)\n")
		fmt.Fprintf(os.Stderr, "  tortise pack -d \">>>\" -o out.tortise *.go  Pack with specific delimiter\n")
		fmt.Fprintf(os.Stderr, "  tortise pack -o out.tortise *.go        Pack with auto-detected delimiter\n")
	}
	
	packFlags.Parse(os.Args[2:])
	
	if packFlags.NArg() < 1 {
		packFlags.Usage()
		os.Exit(1)
	}
	
	var doc *tortise_go.TortiseDocument
	var err error
	
	if packFlags.NArg() == 1 {
		path := packFlags.Arg(0)
		if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
			doc, err = tortise_go.ReadDirectoryTree(path)
		} else {
			doc, err = tortise_go.ReadFiles([]string{path})
		}
	} else {
		filePaths := make([]string, packFlags.NArg())
		for i := 0; i < packFlags.NArg(); i++ {
			filePaths[i] = packFlags.Arg(i)
		}
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