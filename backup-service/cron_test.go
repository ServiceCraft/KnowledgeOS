package main

import (
	"testing"
	"time"
)

func mustParse(t *testing.T, spec string) *cronSchedule {
	t.Helper()
	s, err := parseCron(spec)
	if err != nil {
		t.Fatalf("parseCron(%q) error: %v", spec, err)
	}
	return s
}

func at(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02 15:04 MST", s+" UTC")
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return tm
}

func TestCronDefault0300(t *testing.T) {
	s := mustParse(t, "0 3 * * *")
	if !s.Matches(at(t, "2026-06-07 03:00")) {
		t.Error("expected match at 03:00")
	}
	if s.Matches(at(t, "2026-06-07 03:01")) {
		t.Error("unexpected match at 03:01")
	}
	if s.Matches(at(t, "2026-06-07 02:00")) {
		t.Error("unexpected match at 02:00")
	}
}

func TestCronStepAndRangeAndList(t *testing.T) {
	s := mustParse(t, "*/15 9-17 * * *")
	if !s.Matches(at(t, "2026-06-07 09:00")) || !s.Matches(at(t, "2026-06-07 17:45")) {
		t.Error("expected step/range matches")
	}
	if s.Matches(at(t, "2026-06-07 08:00")) || s.Matches(at(t, "2026-06-07 09:10")) {
		t.Error("unexpected match outside step/range")
	}

	l := mustParse(t, "0 0,12 * * *")
	if !l.Matches(at(t, "2026-06-07 00:00")) || !l.Matches(at(t, "2026-06-07 12:00")) {
		t.Error("expected list match")
	}
	if l.Matches(at(t, "2026-06-07 06:00")) {
		t.Error("unexpected list match")
	}

	// "N/step" means N..max/step (Vixie): "5/15" -> 5,20,35,50.
	ns := mustParse(t, "5/15 * * * *")
	for _, m := range []string{"00:05", "00:20", "00:35", "00:50"} {
		if !ns.Matches(at(t, "2026-06-07 "+m)) {
			t.Errorf("expected N/step match at %s", m)
		}
	}
	if ns.Matches(at(t, "2026-06-07 00:00")) || ns.Matches(at(t, "2026-06-07 00:10")) {
		t.Error("unexpected N/step match")
	}
}

func TestCronDowSevenIsSunday(t *testing.T) {
	// 2026-06-07 is a Sunday.
	sunday := at(t, "2026-06-07 03:00")
	monday := at(t, "2026-06-08 03:00")

	for _, spec := range []string{"0 3 * * 0", "0 3 * * 7"} {
		s := mustParse(t, spec)
		if !s.Matches(sunday) {
			t.Errorf("%q should match Sunday", spec)
		}
		if s.Matches(monday) {
			t.Errorf("%q should not match Monday", spec)
		}
	}

	// Range including 7 must remain valid and match the whole week.
	full := mustParse(t, "0 3 * * 1-7")
	if !full.Matches(sunday) || !full.Matches(monday) {
		t.Error("1-7 dow range should cover Sunday and Monday")
	}
}

func TestCronDomDowOrSemantics(t *testing.T) {
	// When both dom and dow are restricted, match on EITHER (Vixie semantics).
	// 2026-06-07 is Sunday (dow=0), day-of-month 7.
	s := mustParse(t, "0 3 13 * 0") // 13th OR Sunday
	if !s.Matches(at(t, "2026-06-07 03:00")) {
		t.Error("should match Sunday via dow")
	}
	if !s.Matches(at(t, "2026-06-13 03:00")) { // 2026-06-13 is Saturday, dom=13
		t.Error("should match the 13th via dom")
	}
	if s.Matches(at(t, "2026-06-09 03:00")) { // Tuesday, dom=9
		t.Error("should not match a non-13th weekday")
	}
}

func TestCronInvalid(t *testing.T) {
	for _, spec := range []string{"0 3 * *", "60 3 * * *", "0 24 * * *", "0 3 0 * *", "0 3 * 13 *", "a b c d e"} {
		if _, err := parseCron(spec); err == nil {
			t.Errorf("expected error for %q", spec)
		}
	}
}
