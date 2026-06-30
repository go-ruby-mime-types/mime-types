// Copyright (c) the go-ruby-mime-types/mime-types authors
//
// SPDX-License-Identifier: BSD-3-Clause

package mimetypes

import (
	"encoding/json"
	"os/exec"
	"sort"
	"testing"
)

// rubyBin locates a usable `ruby` with the mime-types gem available, once. The
// oracle tests skip themselves when ruby (or the gem) is absent — the qemu
// cross-arch lanes and the Windows lane — so the deterministic suite alone
// drives the 100% coverage gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	if err := exec.Command(path, "-e", "require 'mime-types'").Run(); err != nil {
		t.Skip("mime-types gem not installed; skipping MRI oracle")
	}
	return path
}

// oracleScript dumps the full MRI MIME::Types registry and a per-extension
// type_for map as a single JSON object. $stdout.binmode keeps Windows text-mode
// from polluting the bytes (the go-ruby-erb lesson), though the oracle only runs
// on the non-Windows lanes.
const oracleScript = `
$stdout.binmode
require 'json'
require 'mime-types'
types = MIME::Types.map { |t|
  fr = (t.friendly('en') rescue nil)
  {
    'ct'=>t.content_type, 'media'=>t.media_type, 'sub'=>t.sub_type,
    'ext'=>t.extensions, 'pext'=>t.preferred_extension, 'fr'=>fr,
    'enc'=>t.encoding, 'bin'=>t.binary?, 'ascii'=>t.ascii?,
    'reg'=>t.registered?, 'obs'=>t.obsolete?, 'complete'=>t.complete?,
    'use'=>t.use_instead, 'simplified'=>t.simplified, 'spri'=>t.__sort_priority
  }
}
exts = MIME::Types.map(&:extensions).flatten.uniq.sort
type_for = {}
exts.each { |e| type_for[e] = MIME::Types.type_for("file.#{e}").map(&:to_s) }
print JSON.generate({'count'=>MIME::Types.count, 'types'=>types, 'type_for'=>type_for})
`

type oracleType struct {
	CT         string   `json:"ct"`
	Media      string   `json:"media"`
	Sub        string   `json:"sub"`
	Ext        []string `json:"ext"`
	Pext       string   `json:"pext"`
	Fr         *string  `json:"fr"`
	Enc        string   `json:"enc"`
	Bin        bool     `json:"bin"`
	ASCII      bool     `json:"ascii"`
	Reg        bool     `json:"reg"`
	Obs        bool     `json:"obs"`
	Complete   bool     `json:"complete"`
	Use        *string  `json:"use"`
	Simplified string   `json:"simplified"`
	Spri       int      `json:"spri"`
}

type oracleDump struct {
	Count   int                 `json:"count"`
	Types   []oracleType        `json:"types"`
	TypeFor map[string][]string `json:"type_for"`
}

func loadOracle(t *testing.T) oracleDump {
	t.Helper()
	bin := rubyBin(t)
	out, err := exec.Command(bin, "-e", oracleScript).Output()
	if err != nil {
		t.Fatalf("ruby oracle error: %v", err)
	}
	var d oracleDump
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatalf("decode oracle: %v", err)
	}
	return d
}

// TestOracleRegistryFields checks that every type variant in MRI's registry is
// present in our registry with identical fields (media/sub/extensions/preferred/
// friendly/encoding/binary/ascii/registered/obsolete/complete/use-instead/
// simplified and the precomputed sort priority).
func TestOracleRegistryFields(t *testing.T) {
	d := loadOracle(t)
	r := Default()

	if r.Len() != d.Count {
		t.Fatalf("registry size: got %d, want %d", r.Len(), d.Count)
	}
	if len(r.all) != len(d.Types) {
		t.Fatalf("type count: got %d, want %d", len(r.all), len(d.Types))
	}

	// Index our types by content type for a 1:1 comparison in registry order.
	for i, ot := range d.Types {
		gt := r.all[i]
		if gt.ContentType() != ot.CT {
			t.Fatalf("entry %d: content type got %q, want %q", i, gt.ContentType(), ot.CT)
		}
		if gt.MediaType() != ot.Media {
			t.Errorf("%s: media got %q want %q", ot.CT, gt.MediaType(), ot.Media)
		}
		if gt.SubType() != ot.Sub {
			t.Errorf("%s: sub got %q want %q", ot.CT, gt.SubType(), ot.Sub)
		}
		if !eqStrings(gt.Extensions(), ot.Ext) {
			t.Errorf("%s: ext got %v want %v", ot.CT, gt.Extensions(), ot.Ext)
		}
		if gt.PreferredExtension() != ot.Pext {
			t.Errorf("%s: pref got %q want %q", ot.CT, gt.PreferredExtension(), ot.Pext)
		}
		wantFr := ""
		if ot.Fr != nil {
			wantFr = *ot.Fr
		}
		if gt.Friendly() != wantFr {
			t.Errorf("%s: friendly got %q want %q", ot.CT, gt.Friendly(), wantFr)
		}
		if gt.Encoding() != ot.Enc {
			t.Errorf("%s: encoding got %q want %q", ot.CT, gt.Encoding(), ot.Enc)
		}
		if gt.Binary() != ot.Bin {
			t.Errorf("%s: binary got %v want %v", ot.CT, gt.Binary(), ot.Bin)
		}
		if gt.ASCII() != ot.ASCII {
			t.Errorf("%s: ascii got %v want %v", ot.CT, gt.ASCII(), ot.ASCII)
		}
		if gt.Registered() != ot.Reg {
			t.Errorf("%s: registered got %v want %v", ot.CT, gt.Registered(), ot.Reg)
		}
		if gt.Obsolete() != ot.Obs {
			t.Errorf("%s: obsolete got %v want %v", ot.CT, gt.Obsolete(), ot.Obs)
		}
		if gt.Complete() != ot.Complete {
			t.Errorf("%s: complete got %v want %v", ot.CT, gt.Complete(), ot.Complete)
		}
		wantUse := ""
		if ot.Use != nil {
			wantUse = *ot.Use
		}
		if gt.UseInstead() != wantUse {
			t.Errorf("%s: use-instead got %q want %q", ot.CT, gt.UseInstead(), wantUse)
		}
		if gt.Simplified() != ot.Simplified {
			t.Errorf("%s: simplified got %q want %q", ot.CT, gt.Simplified(), ot.Simplified)
		}
		if gt.sortPriority != ot.Spri {
			t.Errorf("%s: sort-priority got %d want %d", ot.CT, gt.sortPriority, ot.Spri)
		}
	}
}

// TestOracleTypeForAllExtensions checks TypeFor parity against MRI for every
// extension the registry knows, including the priority ordering.
func TestOracleTypeForAllExtensions(t *testing.T) {
	d := loadOracle(t)
	r := Default()

	exts := make([]string, 0, len(d.TypeFor))
	for e := range d.TypeFor {
		exts = append(exts, e)
	}
	sort.Strings(exts)

	for _, ext := range exts {
		want := d.TypeFor[ext]
		got := stringsOf(r.TypeFor("file." + ext))
		if !eqStrings(got, want) {
			t.Errorf("type_for .%s: got %v, want %v", ext, got, want)
		}
	}
}

// TestOracleGetParity spot-checks Get / GetRegexp against MRI for content types
// and patterns that exercise multiple variants and priority ordering.
func TestOracleGetParity(t *testing.T) {
	bin := rubyBin(t)
	r := Default()

	get := func(arg string) []string {
		script := `$stdout.binmode; require 'json'; require 'mime-types'; print JSON.generate(MIME::Types[ARGV[0]].map(&:to_s))`
		out, err := exec.Command(bin, "-e", script, arg).Output()
		if err != nil {
			t.Fatalf("ruby get %q: %v", arg, err)
		}
		var s []string
		if err := json.Unmarshal(out, &s); err != nil {
			t.Fatalf("decode get %q: %v", arg, err)
		}
		return s
	}

	for _, ct := range []string{"text/html", "TEXT/HTML", "application/json", "image/png", "application/gzip", "application/x-tar", "application/access"} {
		want := get(ct)
		got := stringsOf(r.Get(ct))
		if !eqStrings(got, want) {
			t.Errorf("Get(%q): got %v, want %v", ct, got, want)
		}
	}
}

func stringsOf(ts []*Type) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.String())
	}
	return out
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
