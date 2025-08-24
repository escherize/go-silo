# silo 🌾

Pack and unpack directory trees and files into a single text format. Store your code harvest in a silo for later planting! Perfect for sharing project structures and understanding how file paths relate to each other.

<img src="assets/silo.png" alt="silo" width="300">

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

Pack these files into your silo:
```bash
silo pack main.py src/helpers/utils.py -o project.silo
```

This creates a **project.silo** file:
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
silo unpack project.silo
```

The delimiter (`>` in this example) is automatically chosen to avoid conflicts with your file content. Or you can choose one with `-d` using any Unicode character including emojis like `🌾` (see [Silo File Format Spec](https://github.com/escherize/silo_spec)).

# Install

```bash
go install github.com/escherize/silo/cmd/silo@latest
```

# Pack (Harvest files into silo)

```bash
silo pack src/ -o harvest.silo
```

Multiple patterns:
```bash
silo pack "*.go" "*.md" "docs/*.txt"
```

Enhanced recursive harvest:
```bash
silo pack -enhanced "src/**/*.go" -o deep_harvest.silo
```

To stdout:
``` bash
silo pack file1.go file2.go
```

## Custom delimiter (including emojis!)

```bash
silo pack -d "🌾" src/ -o wheat_harvest.silo
```

# Unpack (Plant files from silo)

To current directory:
```bash
silo unpack project.silo
```

To the `field` directory:
```bash
silo unpack project.silo -o field/
```

## Format

A silo file contains multiple files separated by delimiters:
```
🌾 path/to/file1.txt
file1 content here

🌾 path/to/file2.txt
file2 content here
```

When packing, the delimiter (`🌾` in this example) is auto-detected to avoid conflicts with file content. When unpacking, the first delimiter found should be used for every file path.

## Security Features 🔒

Silo protects against path traversal attacks:
- ❌ `../` parent directory references 
- ❌ `/etc/passwd` absolute paths
- ❌ `C:\Windows\System32` drive letters
- ❌ URL-encoded attacks like `%2e%2e%2f`

Only relative paths within your project are allowed!

## Spec

Full specification: https://github.com/escherize/silo-spec
