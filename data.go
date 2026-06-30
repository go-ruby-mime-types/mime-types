// Copyright (c) the go-ruby-mime-types/mime-types authors
//
// SPDX-License-Identifier: BSD-3-Clause

package mimetypes

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"
)

// DataVersion is the version of the embedded mime-types-data registry.
const DataVersion = "3.2026.0414"

// embeddedRegistry is the trimmed mime-types-data dataset, committed in-repo so
// no network or Ruby runtime is needed. It is a JSON array of entries in the
// upstream registry order.
//
//go:embed mimetypes.json
var embeddedRegistry []byte

// rawEntry is the on-disk shape of one embedded registry entry. Field names are
// kept short to minimise the embedded blob.
type rawEntry struct {
	CT   string   `json:"ct"`             // content type
	Enc  string   `json:"enc,omitempty"`  // encoding
	Ext  []string `json:"ext,omitempty"`  // extensions
	Pext string   `json:"pext,omitempty"` // preferred extension
	Fr   string   `json:"fr,omitempty"`   // friendly (en)
	Reg  bool     `json:"reg,omitempty"`  // registered
	Prov bool     `json:"prov,omitempty"` // provisional
	Obs  bool     `json:"obs,omitempty"`  // obsolete
	Sig  bool     `json:"sig,omitempty"`  // signature
	Use  string   `json:"use,omitempty"`  // use-instead
	Docs string   `json:"docs,omitempty"` // docs
}

// mediaTypeRE matches a "media/sub" content type, mirroring the gem's
// MEDIA_TYPE_RE (RFC 6838 §4.2 restricted names).
var mediaTypeRE = regexp.MustCompile(`^([0-9a-zA-Z][-!#$&^_.+0-9a-zA-Z]{0,126})/([0-9a-zA-Z][-!#$&^_.+0-9a-zA-Z]{0,126})`)

// buildEmbedded parses the embedded dataset into a complete [Registry]. It
// panics if the embedded JSON is malformed or an entry has an invalid content
// type, since the embedded data is a build-time constant.
func buildEmbedded() *Registry {
	var raws []rawEntry
	if err := json.Unmarshal(embeddedRegistry, &raws); err != nil {
		panic("mimetypes: corrupt embedded registry: " + err.Error())
	}
	types := make([]*Type, 0, len(raws))
	for i := range raws {
		t, err := newTypeFromRaw(&raws[i])
		if err != nil {
			panic("mimetypes: invalid embedded entry " + raws[i].CT + ": " + err.Error())
		}
		types = append(types, t)
	}
	return New(types)
}

// newTypeFromRaw materialises a [Type] from an embedded entry, deriving the
// simplified key, media/sub parts, default encoding and precomputed sort
// priority exactly as the gem does.
func newTypeFromRaw(r *rawEntry) (*Type, error) {
	// The mime-types-data registry's runtime (columnar) form trims surrounding
	// whitespace from the content type; a few JSON entries carry a stray
	// trailing space, so normalise to match MRI's effective value.
	ct := trimSpace(r.CT)
	m := mediaTypeRE.FindStringSubmatch(ct)
	if m == nil {
		return nil, errInvalidContentType
	}
	t := &Type{
		contentType:        ct,
		mediaType:          lower(m[1]),
		subType:            lower(m[2]),
		extensions:         r.Ext,
		preferredExtension: r.Pext,
		friendly:           r.Fr,
		useInstead:         r.Use,
		docs:               r.Docs,
		registered:         r.Reg,
		provisional:        r.Prov,
		obsolete:           r.Obs,
		signature:          r.Sig,
	}
	t.simplified = t.mediaType + "/" + t.subType
	t.encoding = r.Enc
	if t.encoding == "" {
		t.encoding = defaultEncoding(t.mediaType)
	}
	t.sortPriority = computeSortPriority(t)
	return t, nil
}

// defaultEncoding mirrors MIME::Type#default_encoding: text/* defaults to
// quoted-printable, everything else to base64.
func defaultEncoding(mediaType string) string {
	if mediaType == "text" {
		return "quoted-printable"
	}
	return "base64"
}

// errInvalidContentType is returned when a content type does not match the
// media-type grammar.
var errInvalidContentType = invalidContentTypeError{}

type invalidContentTypeError struct{}

func (invalidContentTypeError) Error() string { return "invalid content type" }

// lower lowercases an ASCII media/sub-type token. The grammar restricts these
// tokens to ASCII, so a byte-wise lowercase is correct and avoids a strings
// import here.
func lower(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if b == nil {
				b = []byte(s)
			}
			b[i] = c + ('a' - 'A')
		}
	}
	if b == nil {
		return s
	}
	return string(b)
}

// trimSpace strips leading and trailing ASCII spaces/tabs from a content type.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// sortStable is a thin indirection over sort.SliceStable so registry.go can sort
// without importing the sort package directly.
func sortStable(ts []*Type, less func(i, j int) bool) {
	sort.SliceStable(ts, less)
}
