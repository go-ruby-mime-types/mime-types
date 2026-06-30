// Copyright (c) the go-ruby-mime-types/mime-types authors
//
// SPDX-License-Identifier: BSD-3-Clause

package mimetypes

import (
	"regexp"
	"strings"
	"sync"
)

// Registry is a queryable set of MIME types, mirroring Ruby's MIME::Types.
//
// A Registry is read-only and safe for concurrent use. Use [Default] for the
// complete embedded registry, or [New] to build one from explicit entries.
type Registry struct {
	all          []*Type            // every type, in registry order
	variants     map[string][]*Type // simplified key -> types sharing it
	extIndex     map[string][]*Type // lowercase extension -> types
	variantOrder []string           // simplified keys in first-seen order (deterministic regexp match)
}

// New builds a Registry from the given types. The supplied types are indexed by
// their simplified key and by extension. Insertion order is preserved.
func New(types []*Type) *Registry {
	r := &Registry{
		variants: make(map[string][]*Type),
		extIndex: make(map[string][]*Type),
	}
	for _, t := range types {
		r.all = append(r.all, t)
		if _, seen := r.variants[t.simplified]; !seen {
			r.variantOrder = append(r.variantOrder, t.simplified)
		}
		r.variants[t.simplified] = append(r.variants[t.simplified], t)
		for _, ext := range t.extensions {
			le := strings.ToLower(ext)
			r.extIndex[le] = append(r.extIndex[le], t)
		}
	}
	return r
}

// Len reports the number of type variants in the registry. Mirrors
// MIME::Types#count.
func (r *Registry) Len() int { return len(r.all) }

// Types returns all type variants in registry order.
func (r *Registry) Types() []*Type {
	out := make([]*Type, len(r.all))
	copy(out, r.all)
	return out
}

// Get returns the types whose simplified content type matches typeID,
// priority-sorted. typeID is matched case-insensitively (e.g. "text/html" or
// "TEXT/HTML"). Mirrors MIME::Types#[] with a string argument.
func (r *Registry) Get(typeID string) []*Type {
	key := simplify(typeID)
	matches := r.variants[key]
	out := make([]*Type, len(matches))
	copy(out, matches)
	sortTypes(out)
	return out
}

// GetRegexp returns every type whose simplified content type matches the
// pattern, priority-sorted and de-duplicated. Mirrors MIME::Types#[] with a
// Regexp argument.
func (r *Registry) GetRegexp(pattern *regexp.Regexp) []*Type {
	var out []*Type
	for _, key := range r.variantOrder {
		if pattern.MatchString(key) {
			out = append(out, r.variants[key]...)
		}
	}
	sortTypes(out)
	return out
}

// extRE extracts the trailing extension the way the gem does:
// fn.chomp.downcase[/\.?([^.]*?)\z/m, 1]. For "a.tar.gz" this yields "gz"; for
// a bare "README" with no dot it yields "readme".
var extRE = regexp.MustCompile(`(?s)\.?([^.]*?)\z`)

// extractExt returns the matching extension token for a filename, mirroring the
// gem's wanted-extension derivation (chomp trailing newline, downcase, take the
// final dot-separated token).
func extractExt(filename string) string {
	f := strings.TrimRight(filename, "\n")
	f = strings.ToLower(f)
	// extRE always matches (the capture group accepts the empty string), so
	// FindStringSubmatch never returns nil here.
	return extRE.FindStringSubmatch(f)[1]
}

// TypeFor returns the types associated with a filename's extension,
// priority-sorted (preferred-extension types favoured) and de-duplicated.
// The lookup is case-insensitive. Mirrors MIME::Types#type_for.
func (r *Registry) TypeFor(filenames ...string) []*Type {
	wanted := make([]string, 0, len(filenames))
	wantedSet := make(map[string]bool, len(filenames))
	for _, fn := range filenames {
		ext := extractExt(fn)
		wanted = append(wanted, ext)
		wantedSet[ext] = true
	}

	var out []*Type
	seen := make(map[*Type]bool)
	for _, ext := range wanted {
		for _, t := range r.extIndex[ext] {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sortByExtension(out, wantedSet)
	return out
}

// Of is an alias for [Registry.TypeFor], mirroring MIME::Types#of.
func (r *Registry) Of(filenames ...string) []*Type {
	return r.TypeFor(filenames...)
}

// sortByExtension sorts in place using the extension-aware comparison.
func sortByExtension(ts []*Type, wanted map[string]bool) {
	stableSort(ts, func(a, b *Type) int {
		return a.extensionPriorityCompare(b, wanted)
	})
}

// stableSort is a small stable insertion-free wrapper that preserves input
// order for equal elements (matching Ruby Array#sort's effective stability for
// our de-duplicated input, where simplified strings break all remaining ties).
func stableSort(ts []*Type, less func(a, b *Type) int) {
	// Reuse sort.SliceStable via the standard library indirection.
	sortStable(ts, func(i, j int) bool { return less(ts[i], ts[j]) < 0 })
}

// simplify lowercases a content type's media/sub parts (no x-prefix removal),
// mirroring MIME::Type.simplified for keying.
func simplify(contentType string) string {
	m := mediaTypeRE.FindStringSubmatch(contentType)
	if m == nil {
		return strings.ToLower(contentType)
	}
	return strings.ToLower(m[1]) + "/" + strings.ToLower(m[2])
}

// Default returns the complete registry built from the embedded
// mime-types-data dataset. It is parsed once on first use.
func Default() *Registry {
	defaultOnce.Do(func() {
		defaultRegistry = buildEmbedded()
	})
	return defaultRegistry
}

var (
	defaultOnce     sync.Once
	defaultRegistry *Registry
)
