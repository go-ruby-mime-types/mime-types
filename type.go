// Copyright (c) the go-ruby-mime-types/mime-types authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package mimetypes is a pure-Go (CGO-free) reimplementation of Ruby's
// MIME::Types registry — the data model and lookup behaviour of the
// mime-types gem (backed by the mime-types-data registry), faithful to MRI.
//
// The complete IANA/registry dataset shipped by the mime-types-data gem is
// embedded (via go:embed of mimetypes.json), so the registry is complete and
// no network or Ruby runtime is required. A [Registry] answers the same
// queries the gem exposes on the MIME::Types module:
//
//	r := mimetypes.Default()
//	r.Get("text/html")          // []*Type, priority sorted
//	r.GetRegexp(re)             // []*Type matching a content-type pattern
//	r.TypeFor("index.html")     // []*Type by filename extension
//	r.Of("a.tar.gz")            // alias of TypeFor
//
// Each [Type] mirrors MIME::Type: ContentType, MediaType, SubType, Extensions,
// PreferredExtension, Friendly, Binary/ASCII, Obsolete, Registered, plus the
// gem's priority ordering (<=>). The value model is what go-embedded-ruby maps
// to Ruby MIME::Type objects.
package mimetypes

import (
	"sort"
	"strings"
)

// Type is a single MIME type entry, mirroring Ruby's MIME::Type.
//
// A Type is immutable once built by the registry; the accessor methods return
// copies of slices where mutation would otherwise leak into the registry.
type Type struct {
	contentType        string   // e.g. "text/HTML" — the raw content type
	mediaType          string   // simplified media type, e.g. "text"
	subType            string   // simplified sub type, e.g. "html"
	simplified         string   // lowercased "media/sub"
	encoding           string   // "base64" | "8bit" | "7bit" | "quoted-printable"
	extensions         []string // file extensions, no leading dot
	preferredExtension string   // explicit preferred extension, or "" (falls back to first)
	friendly           string   // human-readable English description, or ""
	useInstead         string   // replacement content type for obsolete entries, or ""
	docs               string   // documentation note, or ""
	registered         bool     // IANA-registered
	provisional        bool     // provisionally registered
	obsolete           bool     // obsolete entry
	signature          bool     // signature ("magic") type
	sortPriority       int      // precomputed priority (lower sorts first)
}

// binaryEncodings and asciiEncodings mirror MIME::Type::BINARY_ENCODINGS /
// ASCII_ENCODINGS.
var binaryEncodings = map[string]bool{"base64": true, "8bit": true}
var asciiEncodings = map[string]bool{"7bit": true, "quoted-printable": true}

// ContentType returns the raw content type, preserving the registry's casing
// (e.g. "text/html"). Mirrors MIME::Type#content_type.
func (t *Type) ContentType() string { return t.contentType }

// MediaType returns the simplified, lowercased media type (e.g. "text").
// Mirrors MIME::Type#media_type.
func (t *Type) MediaType() string { return t.mediaType }

// SubType returns the simplified, lowercased sub type (e.g. "html").
// Mirrors MIME::Type#sub_type.
func (t *Type) SubType() string { return t.subType }

// Simplified returns the lowercased "media/sub" key. Mirrors
// MIME::Type#simplified.
func (t *Type) Simplified() string { return t.simplified }

// Encoding returns the transfer encoding ("base64", "8bit", "7bit" or
// "quoted-printable"). Mirrors MIME::Type#encoding.
func (t *Type) Encoding() string { return t.encoding }

// Extensions returns the file extensions associated with this type (without a
// leading dot). Mirrors MIME::Type#extensions.
func (t *Type) Extensions() []string {
	out := make([]string, len(t.extensions))
	copy(out, t.extensions)
	return out
}

// PreferredExtension returns the preferred file extension. If none was set
// explicitly, the first extension is used; if there are no extensions, "" is
// returned. Mirrors MIME::Type#preferred_extension.
func (t *Type) PreferredExtension() string {
	if t.preferredExtension != "" {
		return t.preferredExtension
	}
	if len(t.extensions) > 0 {
		return t.extensions[0]
	}
	return ""
}

// Friendly returns the human-readable English description, or "" if none.
// Mirrors MIME::Type#friendly (the "en" entry).
func (t *Type) Friendly() string { return t.friendly }

// UseInstead returns the replacement content type for an obsolete entry, or ""
// if there is none. Mirrors MIME::Type#use_instead.
func (t *Type) UseInstead() string { return t.useInstead }

// Docs returns the documentation note for this type, or "". Mirrors
// MIME::Type#docs.
func (t *Type) Docs() string { return t.docs }

// Registered reports whether the type is IANA-registered. Mirrors
// MIME::Type#registered?.
func (t *Type) Registered() bool { return t.registered }

// Provisional reports whether the type is provisionally registered. Mirrors
// MIME::Type#provisional?.
func (t *Type) Provisional() bool { return t.provisional }

// Obsolete reports whether the type is obsolete. Mirrors MIME::Type#obsolete?.
func (t *Type) Obsolete() bool { return t.obsolete }

// Signature reports whether the type is a signature ("magic") type. Mirrors
// MIME::Type#signature?.
func (t *Type) Signature() bool { return t.signature }

// Complete reports whether the type carries an extension list. Mirrors
// MIME::Type#complete?.
func (t *Type) Complete() bool { return len(t.extensions) > 0 }

// Binary reports whether the type is transferred as binary (base64 or 8bit).
// Mirrors MIME::Type#binary?.
func (t *Type) Binary() bool { return binaryEncodings[t.encoding] }

// ASCII reports whether the type is transferred as ASCII (7bit or
// quoted-printable). Mirrors MIME::Type#ascii?.
func (t *Type) ASCII() bool { return asciiEncodings[t.encoding] }

// String returns the content type, matching MIME::Type#to_s.
func (t *Type) String() string { return t.contentType }

// PriorityCompare orders two types the way MIME::Type#priority_compare (and
// the gem's #<=>) does: by the precomputed sort priority, then by the
// simplified representation alphabetically. It returns -1, 0 or 1.
func (t *Type) PriorityCompare(other *Type) int {
	if c := cmpInt(t.sortPriority, other.sortPriority); c != 0 {
		return c
	}
	return strings.Compare(t.simplified, other.simplified)
}

// extensionPriorityCompare mirrors MIME::Type#__extension_priority_compare:
// when one of the wanted extensions is a type's preferred extension, that
// type's "extension count" bit (0b1000) is cleared to favour it, then the
// usual priority compare applies.
func (t *Type) extensionPriorityCompare(other *Type, wanted map[string]bool) int {
	tsp := t.sortPriority
	if wanted[t.PreferredExtension()] && tsp&0b1000 != 0 {
		tsp = tsp&0b11110111 | 0b0111
	}
	osp := other.sortPriority
	if wanted[other.PreferredExtension()] && osp&0b1000 != 0 {
		osp = osp&0b11110111 | 0b0111
	}
	if c := cmpInt(tsp, osp); c != 0 {
		return c
	}
	return strings.Compare(t.simplified, other.simplified)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// computeSortPriority reproduces MIME::Type#update_sort_priority. Lower numbers
// sort first; the best priority is 0.
//
//	bit 7  obsolete     (1 if true)
//	bit 6  provisional  (1 if true)
//	bit 5  registered   (0 if true, else 1)
//	bit 4  complete     (0 if it has extensions, else 1)
//	bits 0-3  max(0, 16 - extension count)
func computeSortPriority(t *Type) int {
	extCount := len(t.extensions)
	p := 0
	if t.obsolete {
		p |= 1 << 7
	}
	if t.provisional {
		p |= 1 << 6
	}
	if !t.registered {
		p |= 1 << 5
	}
	if extCount == 0 {
		p |= 1 << 4
	}
	rem := 16 - extCount
	if rem < 0 {
		rem = 0
	}
	return p | rem
}

// sortTypes priority-sorts a slice of types in place (MIME::Type#<=>).
func sortTypes(ts []*Type) {
	sort.SliceStable(ts, func(i, j int) bool {
		return ts[i].PriorityCompare(ts[j]) < 0
	})
}
