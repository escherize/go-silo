# tortise

Pack and unpack directory trees and files into a single text format. This is a good way to jot down how file paths are related to each other, like when setting up a project.

<img src="assets/tortise.png" alt="tortise" width="300">

## Example

Let's say you have a Python project with these files:

**main.py**
```python
from src.helpers.utils import add

print(add(2, 3))
```

**src/helpers/utils.py**
```python
def add(a, b):
    return a + b
```

Pack these files:
```bash
tortise pack main.py src/helpers/utils.py -o project.tortise
```

This creates a **project.tortise** file:
```
> main.py
from src.helpers.utils import add

print(add(2, 3))

> src/helpers/utils.py
def add(a, b):
    return a + b
```

Unpack anywhere:
```bash
tortise unpack project.tortise
```

The delimiter (`>` in this example) is automatically chosen to avoid conflicts with your file content. Or you can choose one with `-d` using any punctuation symbol (see [valid delimiters](https://github.com/escherize/tortise_spec#13-abnf-informative)).

# Install

```bash
go install github.com/escherize/tortise_go/cmd/tortise@latest
```

# Pack

```bash
tortise pack src/ -o my_source.tortise
```

To stdout:
``` bash
tortise pack file1.go file2.go
```

## Custom delimiter

```bash
tortise pack -d ">>>" src/ -o output.tortise
```

# Unpack

To this directory:
```bash
tortise unpack project.tortise
```

To the `output` directory
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

When `pack`ing, The delimiter (`>` in this example) is auto-detected to avoid conflicts with file content. When unpacking, the first delimiter found should be used for every file path.

## Spec

Full specification: https://github.com/escherize/tortise_spec
