package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cronSchedule is a parsed standard 5-field cron expression:
//
//	minute hour day-of-month month day-of-week
//
// Supported per field: "*", "*/n", ranges "a-b", lists "a,b", steps "a-b/n",
// and single values. Day-of-week accepts 0 or 7 for Sunday.
type cronSchedule struct {
	minute [60]bool
	hour   [24]bool
	dom    [32]bool // 1..31
	month  [13]bool // 1..12
	dow    [7]bool  // 0..6 (Sunday=0)
	domAny bool     // true when the day-of-month field is "*"
	dowAny bool     // true when the day-of-week field is "*"
}

func parseCron(spec string) (*cronSchedule, error) {
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron must have 5 fields, got %d in %q", len(fields), spec)
	}

	s := &cronSchedule{}
	if err := parseField(fields[0], 0, 59, s.minute[:]); err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	if err := parseField(fields[1], 0, 23, s.hour[:]); err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	if err := parseField(fields[2], 1, 31, s.dom[:]); err != nil {
		return nil, fmt.Errorf("day-of-month: %w", err)
	}
	if err := parseField(fields[3], 1, 12, s.month[:]); err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}

	// Day-of-week accepts both 0 and 7 for Sunday. Parse into an 8-wide set
	// (0..7) so values like "1-7" or "*/7" stay valid, then fold 7 into 0.
	var dowTmp [8]bool
	if err := parseField(fields[4], 0, 7, dowTmp[:]); err != nil {
		return nil, fmt.Errorf("day-of-week: %w", err)
	}
	copy(s.dow[:], dowTmp[:7])
	if dowTmp[7] {
		s.dow[0] = true
	}

	s.domAny = fields[2] == "*"
	s.dowAny = fields[4] == "*"
	return s, nil
}

func parseField(field string, min, max int, set []bool) error {
	for _, part := range strings.Split(field, ",") {
		step := 1
		rangePart := part
		if slash := strings.Index(part, "/"); slash >= 0 {
			rangePart = part[:slash]
			n, err := strconv.Atoi(part[slash+1:])
			if err != nil || n < 1 {
				return fmt.Errorf("invalid step in %q", part)
			}
			step = n
		}

		lo, hi := min, max
		if rangePart != "*" {
			if dash := strings.Index(rangePart, "-"); dash >= 0 {
				a, err1 := strconv.Atoi(rangePart[:dash])
				b, err2 := strconv.Atoi(rangePart[dash+1:])
				if err1 != nil || err2 != nil {
					return fmt.Errorf("invalid range in %q", part)
				}
				lo, hi = a, b
			} else {
				v, err := strconv.Atoi(rangePart)
				if err != nil {
					return fmt.Errorf("invalid value in %q", part)
				}
				// "N/step" means N..max/step (Vixie cron); a bare "N" means just N.
				if step > 1 {
					lo, hi = v, max
				} else {
					lo, hi = v, v
				}
			}
		}

		if lo < min || hi > max || lo > hi {
			return fmt.Errorf("value out of range [%d,%d] in %q", min, max, part)
		}
		for i := lo; i <= hi; i += step {
			set[i] = true
		}
	}
	return nil
}

// Matches reports whether t (interpreted in its own location) satisfies the
// schedule. Following Vixie cron semantics, when both day-of-month and
// day-of-week are restricted, a match on either is sufficient.
func (s *cronSchedule) Matches(t time.Time) bool {
	if !s.minute[t.Minute()] || !s.hour[t.Hour()] || !s.month[int(t.Month())] {
		return false
	}
	domMatch := s.dom[t.Day()]
	dowMatch := s.dow[int(t.Weekday())]

	switch {
	case s.domAny && s.dowAny:
		return true
	case s.domAny:
		return dowMatch
	case s.dowAny:
		return domMatch
	default:
		return domMatch || dowMatch
	}
}
