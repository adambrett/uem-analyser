# Universal Eating Monitor Analyser

Universal Eating Monitor Analyser is a browser-only tool for converting
Universal Eating Monitor text exports into participant Excel workbooks with take,
refill, pause, minute, quartile, and VAS analysis. It ports my old PHP parser
and spreadsheet workflow into reusable Go packages plus a small
WebAssembly-powered single page app.

The app runs entirely in the browser. Uploaded files are parsed locally, VAS
question codes are selected locally, and the generated `.xlsx` or `.zip` download
is created without a server.

## Output

Each participant gets one `.xlsx` workbook. If an upload contains files for more
than one participant, the browser downloads a `.zip` containing those workbooks.

Inside each workbook, every uploaded session file becomes its own worksheet. Each
worksheet contains the legacy report sections: takes, refills, pauses, minute
summaries, quartile summaries, raw VAS results, VAS interpolation by consumed
quartile, and VAS interpolation by elapsed minute.

## Packages

- `pkg/parser` parses Universal Eating Monitor data into typed sessions, takes,
  refills, pauses, and VAS results.
- `internal/analyser` validates uploaded files, detects VAS codes, groups sessions
  by participant, and creates the final download payload.
- `internal/spreadsheet` writes behaviour-compatible XLSX workbooks with the
  legacy report sections.
- `cmd/uem-analyser` exposes the Go processing pipeline to the browser as
  WebAssembly.

## Development

Run the test suite:

```sh
make test
```

Run the WebAssembly-targeted tests:

```sh
make test-wasm
```

Build the static site into `dist/`:

```sh
make build
```

Serve the built site locally:

```sh
make serve
```

## Requirements

- Go 1.26.2 or compatible
- Node.js available to Go's `go_js_wasm_exec` wrapper for `make test-wasm`
- Python 3 for `make serve`

## Contributing

### Pull Requests

1. Fork the repository.
2. Create a new branch for each feature or improvement.
3. Send a pull request from each feature branch.

### Style Guide

This project is formatted with gofmt using `make fmt` and follows idiomatic Go
conventions. If you notice style or API oversights, please send a patch via pull
request.

### Tests

The library is developed using test driven development. All pull requests should
be accompanied by passing unit tests with 100% coverage. Go's `testing` package
and Testify are used for testing and assertions.

## License

This project is licensed under BSD-3-Clause. See [LICENSE](LICENSE).
