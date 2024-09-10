RFC 9535 JSONPath in Go
=======================

[![⚖️ MIT]][mit] [![📚 Docs]][docs] [![🗃️ Report Card]][card] [![🛠️ Build Status]][ci] [![📊 Coverage]][cov]

The jsonpath package provides [RFC 9535 JSONPath] functionality in Go.

## Project Status

While the implementation is complete, this package is currently under active
development, mainly refactoring, reorganizing, renaming, and documenting.
However, the public interface of the `jsonpath` package is stable, though new
methods will be added. Only sub-packages are actively in flux.

## Copyright

Copyright © 2024 David E. Wheeler

  [⚖️ MIT]: https://img.shields.io/badge/License-MIT-blue.svg "⚖️ MIT License"
  [mit]: https://opensource.org/license/MIT "⚖️ MIT License"
  [📚 Docs]: https://godoc.org/github.com/theory/jsonpath?status.svg "📚 Documentation"
  [docs]: https://pkg.go.dev/github.com/theory/jsonpath "📄 Documentation"
  [🗃️ Report Card]: https://goreportcard.com/badge/github.com/theory/jsonpath
    "🗃️ Report Card"
  [card]: https://goreportcard.com/report/github.com/theory/jsonpath
    "🗃️ Report Card"
  [🛠️ Build Status]: https://github.com/theory/jsonpath/actions/workflows/ci.yml/badge.svg
    "🛠️ Build Status"
  [ci]: https://github.com/theory/jsonpath/actions/workflows/ci.yml
    "🛠️ Build Status"
  [📊 Coverage]: https://codecov.io/gh/theory/jsonpath/graph/badge.svg?token=UB1UJ95NIK
    "📊 Code Coverage"
  [cov]: https://codecov.io/gh/theory/jsonpath "📊 Code Coverage"
  [RFC 9535 JSONPath]: https://www.rfc-editor.org/rfc/rfc9535.html
    "RFC 9535 JSONPath: Query Expressions for JSON"
