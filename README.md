<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-mime-types/brand/main/social/go-ruby-mime-types-mime-types.png" alt="go-ruby-mime-types/mime-types" width="720"></p>

# mime-types — go-ruby-mime-types

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-mime-types.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's
[`MIME::Types`](https://github.com/mime-types/ruby-mime-types) registry** — the
data model and lookup behaviour of the `mime-types` gem, backed by the complete
[`mime-types-data`](https://github.com/mime-types/mime-types-data) dataset.
Look up MIME types by content type or by filename extension, with the gem's
exact priority ordering — **without any Ruby runtime**.

It is the MIME-type backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) (the Onigmo engine),
[go-ruby-erb](https://github.com/go-ruby-erb/erb) (the ERB compiler) and
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (the Psych emitter/loader).

> **What it is — and isn't.** This is the MIME type *registry and lookup* — a
> deterministic, interpreter-independent value model that needs **no interpreter**,
> so it lives here as pure Go. The full IANA/registry dataset is embedded
> (`go:embed`), so the registry is **complete** and needs no network. Binding the
> returned `*Type` values to live Ruby `MIME::Type` objects is the host's job; this
> library hands back a small, explicit model the host maps to and from its own
> objects.

## Features

Faithful port of `MIME::Types`, validated field-for-field against the
`mime-types` gem across the **entire registry** (every variant and every
extension) on every supported platform:

- **Complete embedded registry** — the `mime-types-data` dataset
  (version `3.2026.0414`, **3094 type variants / 1612 extensions**) is committed
  in-repo and embedded via `go:embed`. No network, no gem, no Ruby.
- **`Get` by content type** (`MIME::Types[str]`) — case-insensitive, returns the
  priority-sorted variants for a simplified `media/sub` key.
- **`GetRegexp` by pattern** (`MIME::Types[/re/]`) — every type whose simplified
  content type matches, priority-sorted and de-duplicated.
- **`TypeFor` / `Of` by filename** (`MIME::Types.type_for` / `.of`) — extension
  lookup (case-insensitive), with the gem's preferred-extension boost and
  multi-filename union.
- **`Type` value model** — `ContentType`, `MediaType`, `SubType`, `Simplified`,
  `Encoding`, `Extensions`, `PreferredExtension`, `Friendly`, `UseInstead`,
  `Docs`, and the `Registered` / `Provisional` / `Obsolete` / `Signature` /
  `Complete` / `Binary` / `ASCII` predicates.
- **Exact priority ordering** — `PriorityCompare` reproduces the gem's
  precomputed sort priority bitmap (obsolete / provisional / registered /
  complete / extension count) and its simplified-name tiebreak.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) on Linux, macOS and Windows.

## Install

```sh
go get github.com/go-ruby-mime-types/mime-types
```

## Usage

```go
package main

import (
	"fmt"
	"regexp"

	mimetypes "github.com/go-ruby-mime-types/mime-types"
)

func main() {
	r := mimetypes.Default() // the complete embedded registry

	// By filename extension — MIME::Types.type_for / .of
	for _, t := range r.TypeFor("report.html") {
		fmt.Println(t) // text/html
	}
	fmt.Println(r.TypeFor("archive.tar.gz")) // [application/gzip application/x-gzip]

	// By content type — MIME::Types[str] (case-insensitive)
	html := r.Get("text/html")[0]
	fmt.Println(html.MediaType(), html.SubType()) // text html
	fmt.Println(html.PreferredExtension())        // html
	fmt.Println(html.Friendly())                  // HyperText Markup Language (HTML)
	fmt.Println(html.Binary(), html.Registered()) // true true

	// By regexp — MIME::Types[/re/]
	for _, t := range r.GetRegexp(regexp.MustCompile(`^image/`)) {
		_ = t
	}
}
```

## Value model

A `*Type` mirrors Ruby's `MIME::Type`; a `*Registry` mirrors the `MIME::Types`
module. A host (such as go-embedded-ruby) maps these onto its own Ruby objects:

| Ruby (`MIME::Type`)         | Go (`*mimetypes.Type`)        |
| --------------------------- | ----------------------------- |
| `#content_type`             | `ContentType() string`        |
| `#media_type`               | `MediaType() string`          |
| `#sub_type`                 | `SubType() string`            |
| `#simplified`               | `Simplified() string`         |
| `#encoding`                 | `Encoding() string`           |
| `#extensions`               | `Extensions() []string`       |
| `#preferred_extension`      | `PreferredExtension() string` |
| `#friendly`                 | `Friendly() string`           |
| `#use_instead`              | `UseInstead() string`         |
| `#docs`                     | `Docs() string`               |
| `#registered?`              | `Registered() bool`           |
| `#provisional?`             | `Provisional() bool`          |
| `#obsolete?`                | `Obsolete() bool`             |
| `#signature?`               | `Signature() bool`            |
| `#complete?`                | `Complete() bool`             |
| `#binary?` / `#ascii?`      | `Binary() bool` / `ASCII() bool` |
| `#<=>` / `#priority_compare`| `PriorityCompare(*Type) int`  |

| Ruby (`MIME::Types`)        | Go (`*mimetypes.Registry`)            |
| --------------------------- | ------------------------------------- |
| `MIME::Types[str]`          | `Get(typeID string) []*Type`          |
| `MIME::Types[/re/]`         | `GetRegexp(*regexp.Regexp) []*Type`   |
| `MIME::Types.type_for(fn)`  | `TypeFor(filenames ...string) []*Type`|
| `MIME::Types.of(fn)`        | `Of(filenames ...string) []*Type`     |
| `MIME::Types.count`         | `Len() int`                           |

## API

```go
// Default returns the complete registry built from the embedded
// mime-types-data dataset (parsed once on first use).
func Default() *Registry

// New builds a Registry from explicit types (insertion order preserved).
func New(types []*Type) *Registry

func (r *Registry) Get(typeID string) []*Type           // MIME::Types[str]
func (r *Registry) GetRegexp(p *regexp.Regexp) []*Type   // MIME::Types[/re/]
func (r *Registry) TypeFor(filenames ...string) []*Type  // MIME::Types.type_for
func (r *Registry) Of(filenames ...string) []*Type       // MIME::Types.of
func (r *Registry) Len() int                             // MIME::Types.count
func (r *Registry) Types() []*Type                       // every variant

func (t *Type) ContentType() string
func (t *Type) MediaType() string
func (t *Type) SubType() string
func (t *Type) Simplified() string
func (t *Type) Encoding() string
func (t *Type) Extensions() []string
func (t *Type) PreferredExtension() string
func (t *Type) Friendly() string
func (t *Type) UseInstead() string
func (t *Type) Docs() string
func (t *Type) Registered() bool
func (t *Type) Provisional() bool
func (t *Type) Obsolete() bool
func (t *Type) Signature() bool
func (t *Type) Complete() bool
func (t *Type) Binary() bool
func (t *Type) ASCII() bool
func (t *Type) String() string
func (t *Type) PriorityCompare(other *Type) int

const DataVersion = "3.2026.0414" // embedded mime-types-data version
```

## Tests & coverage

```sh
go test -race -cover ./...
```

The suite holds **100% statement coverage**. A differential **MRI oracle**
(`oracle_test.go`) shells out to the `mime-types` gem and asserts byte-for-byte
parity across the whole registry — every field of all 3094 variants and the
`type_for` result for all 1612 extensions, including priority ordering. The
oracle skips itself when `ruby` (or the gem) is absent — the Windows and qemu
cross-arch lanes — where the deterministic, Ruby-free tests alone keep coverage
at 100%.

## License

[BSD-3-Clause](LICENSE) © the go-ruby-mime-types/mime-types authors.

The embedded registry data derives from
[`mime-types-data`](https://github.com/mime-types/mime-types-data) (MIT).

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
