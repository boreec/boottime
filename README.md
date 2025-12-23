# bootprobe

## Requirements

- Go
- systemd-analyze

## Usage

Append boot time analysis results to a jsonl file. The file is created if it
does not exist.

```console
go run ./cmd/bootprobe -t -f results.jsonl
```

Display the average of boot time analysis results:

```console
go run ./cmd/bootprobe -A -f results.jsonl
```
