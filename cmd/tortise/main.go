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
	delimiter := packFlags.String("d", ">", "Delimiter to use")
	
	packFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tortise pack [options] <directory>\n")
		fmt.Fprintf(os.Stderr, "Pack a directory tree into a tortise file\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		packFlags.PrintDefaults()
	}
	
	packFlags.Parse(os.Args[2:])
	
	if packFlags.NArg() != 1 {
		packFlags.Usage()
		os.Exit(1)
	}
	
	dirPath := packFlags.Arg(0)
	
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Directory does not exist: %s\n", dirPath)
		os.Exit(1)
	}
	
	doc, err := tortise_go.ReadDirectoryTree(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory tree: %v\n", err)
		os.Exit(1)
	}
	
	doc.Delimiter = *delimiter
	
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
	fmt.Fprintf(os.Stderr, "tortise - A tool for packing/unpacking directory trees\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  tortise pack [options] <directory>     Pack directory into tortise file\n")
	fmt.Fprintf(os.Stderr, "  tortise unpack [options] <file>        Unpack tortise file into directory\n")
	fmt.Fprintf(os.Stderr, "  tortise help                           Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  tortise pack src/ -o project.tortise   Pack 'src' directory\n")
	fmt.Fprintf(os.Stderr, "  tortise pack src/                      Pack to stdout\n")
	fmt.Fprintf(os.Stderr, "  tortise unpack project.tortise         Unpack to current directory\n")
	fmt.Fprintf(os.Stderr, "  tortise unpack project.tortise -o out/ Unpack to 'out' directory\n")
}