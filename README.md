# tortise

Pack and unpack directory trees and files into a single text format.

## Install

```bash
go install github.com/escherize/tortise_go/cmd/tortise@latest
```

## Pack a directory

```bash
tortise pack src/
```

## Pack specific files

```bash
tortise pack file1.go file2.go
```

## Unpack

```bash
tortise unpack project.tortise
```

## Custom delimiter

```bash
tortise pack -d ">>>" src/ -o output.tortise
```

## Output to file

```bash
tortise pack src/ -o project.tortise
```

## Unpack to directory

```bash
tortise unpack project.tortise -o output/
```

## Format

A tortise file contains multiple files separated by delimiters:

```
> path/to/file1.txt
file1 content here

> path/to/file2.txt
file2 content here
```

The delimiter (`>` in this example) is auto-detected to avoid conflicts with file content.

## Spec

Full specification: https://github.com/escherize/tortise_spec