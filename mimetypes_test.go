// Copyright (c) the go-ruby-mime-types/mime-types authors
//
// SPDX-License-Identifier: BSD-3-Clause

package mimetypes

import (
	"regexp"
	"testing"
)

// newTestType builds a Type from a raw entry for unit tests, mirroring the
// embedded loader path.
func newTestType(t *testing.T, r *rawEntry) *Type {
	t.Helper()
	typ, err := newTypeFromRaw(r)
	if err != nil {
		t.Fatalf("newTypeFromRaw(%+v): %v", r, err)
	}
	return typ
}

func TestDataVersion(t *testing.T) {
	if DataVersion == "" {
		t.Fatal("DataVersion is empty")
	}
}

func TestDefaultRegistrySingleton(t *testing.T) {
	a := Default()
	b := Default()
	if a != b {
		t.Fatal("Default() returned distinct registries")
	}
	if a.Len() == 0 {
		t.Fatal("Default() registry is empty")
	}
}

func TestGetCaseInsensitive(t *testing.T) {
	r := Default()
	a := r.Get("text/html")
	b := r.Get("TEXT/HTML")
	if len(a) != 1 || len(b) != 1 || a[0].ContentType() != b[0].ContentType() {
		t.Fatalf("case-insensitive Get mismatch: %v vs %v", stringsOf(a), stringsOf(b))
	}
	if got := r.Get("does/notexist"); len(got) != 0 {
		t.Fatalf("Get for unknown type: %v", stringsOf(got))
	}
}

func TestGetReturnsCopy(t *testing.T) {
	r := Default()
	a := r.Get("text/html")
	a[0] = nil // must not corrupt the registry
	b := r.Get("text/html")
	if len(b) != 1 || b[0] == nil {
		t.Fatal("Get did not return an independent slice")
	}
}

func TestGetRegexp(t *testing.T) {
	r := Default()
	got := r.GetRegexp(regexp.MustCompile(`^text/`))
	if len(got) == 0 {
		t.Fatal("expected text/* matches")
	}
	for _, x := range got {
		if x.MediaType() != "text" {
			t.Fatalf("non-text match: %s", x)
		}
	}
	// Priority ordering: results are sorted, so the first should sort <= the second.
	for i := 1; i < len(got); i++ {
		if got[i-1].PriorityCompare(got[i]) > 0 {
			t.Fatalf("GetRegexp not priority sorted at %d: %s then %s", i, got[i-1], got[i])
		}
	}
	if none := r.GetRegexp(regexp.MustCompile(`zzznomatchzzz`)); len(none) != 0 {
		t.Fatalf("expected no matches, got %v", stringsOf(none))
	}
}

func TestTypeForAndOf(t *testing.T) {
	r := Default()
	cases := []struct {
		file string
		want []string
	}{
		{"index.html", []string{"text/html"}},
		{"data.json", []string{"application/json"}},
		{"pic.PNG", []string{"image/png"}},
		{"archive.tar.gz", []string{"application/gzip", "application/x-gzip"}},
		{"nope.zzzznotanext", nil},
		// A bare filename with no dot is matched as an extension token; "readme"
		// is a registered text/plain extension, matching MRI.
		{"README", []string{"text/plain"}},
	}
	for _, c := range cases {
		got := stringsOf(r.TypeFor(c.file))
		if !eqStrings(got, c.want) {
			t.Errorf("TypeFor(%q) = %v, want %v", c.file, got, c.want)
		}
		// Of is an alias.
		if of := stringsOf(r.Of(c.file)); !eqStrings(of, got) {
			t.Errorf("Of(%q) = %v, want %v", c.file, of, got)
		}
	}
}

func TestTypeForMultipleFilenames(t *testing.T) {
	r := Default()
	got := stringsOf(r.TypeFor("a.html", "b.png"))
	// Both extensions contribute; result is the priority-sorted union, deduped.
	if len(got) < 2 {
		t.Fatalf("expected at least html+png, got %v", got)
	}
	hasHTML, hasPNG := false, false
	for _, s := range got {
		if s == "text/html" {
			hasHTML = true
		}
		if s == "image/png" {
			hasPNG = true
		}
	}
	if !hasHTML || !hasPNG {
		t.Fatalf("missing html/png: %v", got)
	}
}

func TestTypeForDedupesSharedExtension(t *testing.T) {
	r := Default()
	// "gz" and "tgz" both map to application/gzip; passing both filenames must
	// not duplicate it.
	got := stringsOf(r.TypeFor("a.gz", "b.tgz"))
	seen := map[string]int{}
	for _, s := range got {
		seen[s]++
	}
	for s, n := range seen {
		if n > 1 {
			t.Fatalf("%s duplicated %d times in %v", s, n, got)
		}
	}
}

func TestExtractExt(t *testing.T) {
	cases := map[string]string{
		"a.html":     "html",
		"a.tar.gz":   "gz",
		"A.TAR.GZ":   "gz",
		"README":     "readme",
		"trailing\n": "trailing",
		".bashrc":    "bashrc",
		"":           "",
		"a.":         "",
	}
	for in, want := range cases {
		if got := extractExt(in); got != want {
			t.Errorf("extractExt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTypeAccessors(t *testing.T) {
	r := Default()
	h := r.Get("text/html")[0]
	if h.ContentType() != "text/html" || h.MediaType() != "text" || h.SubType() != "html" {
		t.Fatalf("content/media/sub: %s %s %s", h.ContentType(), h.MediaType(), h.SubType())
	}
	if h.Simplified() != "text/html" {
		t.Fatalf("simplified: %s", h.Simplified())
	}
	if h.PreferredExtension() != "html" {
		t.Fatalf("pref ext: %s", h.PreferredExtension())
	}
	if h.Friendly() == "" {
		t.Fatal("expected a friendly name for text/html")
	}
	if !h.Registered() || h.Obsolete() || !h.Complete() {
		t.Fatalf("flags: reg=%v obs=%v complete=%v", h.Registered(), h.Obsolete(), h.Complete())
	}
	if h.String() != h.ContentType() {
		t.Fatal("String should equal ContentType")
	}
	exts := h.Extensions()
	exts[0] = "mutated" // must not affect registry
	if r.Get("text/html")[0].Extensions()[0] == "mutated" {
		t.Fatal("Extensions did not return a copy")
	}
	// Encoding accessor.
	if h.Encoding() == "" {
		t.Fatal("expected an encoding")
	}
}

func TestObsoleteAndUseInstead(t *testing.T) {
	r := Default()
	// application/access is obsolete with a use-instead clause in the registry.
	ts := r.Get("application/access")
	if len(ts) == 0 {
		t.Skip("registry no longer carries application/access")
	}
	o := ts[0]
	if !o.Obsolete() {
		t.Fatal("application/access should be obsolete")
	}
	if o.UseInstead() == "" {
		t.Fatal("expected a use-instead replacement")
	}
}

func TestProvisionalSignatureDocsAccessors(t *testing.T) {
	// Synthetic entries exercise the Provisional/Signature/Docs accessors and
	// the provisional sort-priority bit, which the registry may not surface.
	pt := newTestType(t, &rawEntry{CT: "application/x-prov", Ext: []string{"prv"}, Prov: true})
	if !pt.Provisional() {
		t.Fatal("expected provisional")
	}
	if pt.sortPriority&(1<<6) == 0 {
		t.Fatal("provisional bit not set in sort priority")
	}
	st := newTestType(t, &rawEntry{CT: "application/x-sig", Sig: true, Docs: "see RFC"})
	if !st.Signature() {
		t.Fatal("expected signature")
	}
	if st.Docs() != "see RFC" {
		t.Fatalf("docs: %q", st.Docs())
	}
}

func TestBinaryAsciiAndDefaultEncoding(t *testing.T) {
	// text/* with no explicit encoding defaults to quoted-printable (ascii);
	// other media defaults to base64 (binary). This exercises defaultEncoding.
	txt := newTestType(t, &rawEntry{CT: "text/x-foo"})
	if txt.Encoding() != "quoted-printable" || !txt.ASCII() || txt.Binary() {
		t.Fatalf("text default: enc=%s ascii=%v bin=%v", txt.Encoding(), txt.ASCII(), txt.Binary())
	}
	app := newTestType(t, &rawEntry{CT: "application/x-foo"})
	if app.Encoding() != "base64" || !app.Binary() || app.ASCII() {
		t.Fatalf("app default: enc=%s bin=%v ascii=%v", app.Encoding(), app.Binary(), app.ASCII())
	}
	// 7bit is ascii; 8bit is binary.
	sevenBit := newTestType(t, &rawEntry{CT: "text/x-bar", Enc: "7bit"})
	if !sevenBit.ASCII() || sevenBit.Binary() {
		t.Fatal("7bit should be ascii, not binary")
	}
	eightBit := newTestType(t, &rawEntry{CT: "text/x-baz", Enc: "8bit"})
	if !eightBit.Binary() || eightBit.ASCII() {
		t.Fatal("8bit should be binary, not ascii")
	}
}

func TestPreferredExtensionFallbacks(t *testing.T) {
	// Explicit preferred extension wins.
	explicit := newTestType(t, &rawEntry{CT: "application/x-a", Ext: []string{"aa", "bb"}, Pext: "bb"})
	if explicit.PreferredExtension() != "bb" {
		t.Fatalf("explicit pref: %s", explicit.PreferredExtension())
	}
	// No explicit preferred -> first extension.
	first := newTestType(t, &rawEntry{CT: "application/x-b", Ext: []string{"cc", "dd"}})
	if first.PreferredExtension() != "cc" {
		t.Fatalf("first-ext pref: %s", first.PreferredExtension())
	}
	// No extensions -> "".
	none := newTestType(t, &rawEntry{CT: "application/x-c"})
	if none.PreferredExtension() != "" {
		t.Fatalf("empty pref: %q", none.PreferredExtension())
	}
	if none.Complete() {
		t.Fatal("entry without extensions should be incomplete")
	}
}

func TestPriorityCompare(t *testing.T) {
	// Registered+complete sorts before unregistered/incomplete; ties break on
	// simplified name.
	reg := newTestType(t, &rawEntry{CT: "application/aaa", Ext: []string{"x"}, Reg: true})
	unreg := newTestType(t, &rawEntry{CT: "application/bbb", Ext: []string{"x"}})
	if reg.PriorityCompare(unreg) >= 0 {
		t.Fatal("registered should sort before unregistered")
	}
	if unreg.PriorityCompare(reg) <= 0 {
		t.Fatal("unregistered should sort after registered")
	}
	// Equal priority -> alphabetical by simplified, returns nonzero.
	a := newTestType(t, &rawEntry{CT: "application/aaa", Ext: []string{"x"}, Reg: true})
	b := newTestType(t, &rawEntry{CT: "application/zzz", Ext: []string{"x"}, Reg: true})
	if a.PriorityCompare(b) >= 0 {
		t.Fatal("a should sort before z on simplified tiebreak")
	}
	// Identical simplified -> 0.
	if a.PriorityCompare(a) != 0 {
		t.Fatal("identical types should compare equal")
	}
}

func TestExtensionPriorityCompareTieAndPreferred(t *testing.T) {
	// Two registered types each with a single extension "sx" plus one filler, so
	// both have the same base sort priority (16-2 = 14, with bit 0b1000 set).
	// "pref" lists "sx" first (its preferred extension); "nonpref" lists "sx"
	// second. Under TypeFor("file.sx") the preferred-extension boost clears
	// "pref"'s 0b1000 bit, ordering it ahead of "nonpref" even though their
	// unboosted priorities are identical. This exercises both arms of
	// extensionPriorityCompare and its boost path.
	pref := newTestType(t, &rawEntry{CT: "application/pref", Ext: []string{"sx", "z1"}, Reg: true})
	nonpref := newTestType(t, &rawEntry{CT: "application/nonpref", Ext: []string{"z2", "sx"}, Reg: true})
	r := New([]*Type{nonpref, pref}) // registry order favours nonpref absent the boost

	got := stringsOf(r.TypeFor("file.sx"))
	if len(got) != 2 {
		t.Fatalf("expected both shared types, got %v", got)
	}
	if got[0] != "application/pref" {
		t.Fatalf("preferred-extension boost not applied: %v", got)
	}

	// Direct compare both ways with "sx" wanted: pref (boosted) sorts first.
	wanted := map[string]bool{"sx": true}
	if pref.extensionPriorityCompare(nonpref, wanted) >= 0 {
		t.Fatal("boosted pref should sort before nonpref")
	}
	if nonpref.extensionPriorityCompare(pref, wanted) <= 0 {
		t.Fatal("nonpref should sort after boosted pref")
	}
	// Identical type compared to itself with no boost effect -> simplified tie -> 0.
	if pref.extensionPriorityCompare(pref, wanted) != 0 {
		t.Fatal("identical types should compare equal under extension compare")
	}
}

func TestSimplifyEdgeCases(t *testing.T) {
	if got := simplify("Text/HTML"); got != "text/html" {
		t.Fatalf("simplify cased: %q", got)
	}
	// No media/sub slash -> lowercase fallback.
	if got := simplify("garbage"); got != "garbage" {
		t.Fatalf("simplify fallback: %q", got)
	}
}

func TestNewTypeFromRawInvalid(t *testing.T) {
	if _, err := newTypeFromRaw(&rawEntry{CT: "no-slash-here"}); err == nil {
		t.Fatal("expected error for invalid content type")
	} else if err.Error() != "invalid content type" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTrimSpace(t *testing.T) {
	cases := map[string]string{
		"  a  ":       "a",
		"\ta\t":       "a",
		"application": "application",
		"   ":         "",
		"":            "",
		"a/b ":        "a/b",
	}
	for in, want := range cases {
		if got := trimSpace(in); got != want {
			t.Errorf("trimSpace(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContentTypeTrimmedFromRegistry(t *testing.T) {
	// The registry carries an entry whose raw JSON content type has a trailing
	// space; it must be exposed trimmed, matching MRI.
	r := Default()
	ts := r.Get("application/vnd.java.hprof")
	if len(ts) == 0 {
		t.Skip("hprof entry no longer present")
	}
	if ts[0].ContentType() != "application/vnd.java.hprof" {
		t.Fatalf("content type not trimmed: %q", ts[0].ContentType())
	}
}

func TestTypesSnapshot(t *testing.T) {
	r := Default()
	all := r.Types()
	if len(all) != r.Len() {
		t.Fatalf("Types() length %d != Len() %d", len(all), r.Len())
	}
	all[0] = nil // must not corrupt the registry
	if r.Types()[0] == nil {
		t.Fatal("Types() did not return a copy")
	}
}

func TestLowerHelper(t *testing.T) {
	if lower("ABC") != "abc" {
		t.Fatal("lower failed on uppercase")
	}
	if lower("abc") != "abc" {
		t.Fatal("lower changed all-lowercase input")
	}
	if lower("aBc") != "abc" {
		t.Fatal("lower failed on mixed case")
	}
}

func TestComputeSortPriorityCapsExtensionBits(t *testing.T) {
	// More than 16 extensions clamps the extension contribution to 0.
	exts := make([]string, 20)
	for i := range exts {
		exts[i] = string(rune('a' + i))
	}
	many := newTestType(t, &rawEntry{CT: "application/x-many", Ext: exts, Reg: true})
	if many.sortPriority&0b1111 != 0 {
		t.Fatalf("extension bits not clamped to 0: %04b", many.sortPriority&0b1111)
	}
}

func TestBuildEmbeddedPanicsOnCorruptJSON(t *testing.T) {
	saved := embeddedRegistry
	defer func() {
		embeddedRegistry = saved
		if r := recover(); r == nil {
			t.Fatal("expected panic on corrupt JSON")
		}
	}()
	embeddedRegistry = []byte("{not json")
	buildEmbedded()
}

func TestBuildEmbeddedPanicsOnInvalidEntry(t *testing.T) {
	saved := embeddedRegistry
	defer func() {
		embeddedRegistry = saved
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid entry")
		}
	}()
	embeddedRegistry = []byte(`[{"ct":"no-slash"}]`)
	buildEmbedded()
}
