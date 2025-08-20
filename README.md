# tortise

Pack and unpack directory trees and files into a single text format.

![tortise](assets/tortise.png)

## Example

Let's say you have a Python project with these files:

**main.py**
```python
from utils import add

print(add(2, 3))
```

**utils.py**
```python
def add(a, b):
    return a + b
```

Pack these files:
```bash
tortise pack main.py utils.py -o project.tortise
```

This creates a **project.tortise** file:
```
> main.py
from utils import add

print(add(2, 3))

> utils.py
def add(a, b):
    return a + b
```

Unpack anywhere:
```bash
tortise unpack project.tortise
```

The delimiter (`>` in this example) is automatically chosen to avoid conflicts with your file content. Or you can choose one with `-d` using any punctuation symbol (see [valid delimiters](https://github.com/escherize/tortise_spec#13-abnf-informative)).

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