# TODO

By order of priority.

- [ ] Make boot time retrieving failure from a source as non-blocking for other 
  sources.
- [ ] Refactor exec into better named packages.
- [ ] Cover aggregation and average logic with tests.
- [ ] Check other ACPI tables that could be used.
- [ ] For EFI vars, retrieve the files of interest with binary search.
- [ ] Add pre-commit file.
- [ ] Enable linting with golangci-lint.
- [ ] Run go tests and linting in CICD.

## Done

- [X] Retrieve boot time from `/sys/firmware/acpi/tables/FPDT`.

