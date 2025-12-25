# Boottime

**Boottime** is a tool for Linux OS boot time analysis.

## Requirements

- Go
- systemd
- systemd-analyze (optional)

## Usage

Append boot time analysis results to a jsonl file. The file is created if it
does not exist.

```console
go run ./cmd/boottime -R results.jsonl
```

Display the average of boot time analysis results:

```console
go run ./cmd/boottime -A results.jsonl
```
