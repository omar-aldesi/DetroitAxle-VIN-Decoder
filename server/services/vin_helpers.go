package services

import (
	"strings"
	"time"
)

// coalesce returns the first non-empty trimmed string.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

// nhtsaStr safely reads a field from a possibly-nil NHTSA result.
func nhtsaStr(n *nhtsaResult, get func(*nhtsaResult) string) string {
	if n == nil {
		return ""
	}
	return get(n)
}

// normalizeMake trims the make and fixes ONLY the all-uppercase case
// (e.g. "CHEVROLET" → "Chevrolet"). Mixed-case values from auto.dev such as
// "GMC", "BMW", "Chevrolet" are returned untouched so acronyms stay intact.
func normalizeMake(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s == strings.ToUpper(s) && len(s) > 3 {
		// All caps and longer than a typical acronym → title-case it.
		return strings.Title(strings.ToLower(s))
	}
	return s
}

// vinYearCodes maps the VIN's 10th character to the FIRST model year it
// represents (starting 1980). The code repeats on a 30-year cycle; the letters
// I, O, Q, U, Z and the digit 0 are never used in position 10.
var vinYearCodes = map[byte]int{
	'A': 1980, 'B': 1981, 'C': 1982, 'D': 1983, 'E': 1984, 'F': 1985,
	'G': 1986, 'H': 1987, 'J': 1988, 'K': 1989, 'L': 1990, 'M': 1991,
	'N': 1992, 'P': 1993, 'R': 1994, 'S': 1995, 'T': 1996, 'V': 1997,
	'W': 1998, 'X': 1999, 'Y': 2000,
	'1': 2001, '2': 2002, '3': 2003, '4': 2004, '5': 2005,
	'6': 2006, '7': 2007, '8': 2008, '9': 2009,
}

// modelYearFromVIN decodes the model year from the VIN's 10th character.
// Because the code cycles every 30 years, we pick the most recent candidate
// that is no later than next model year relative to today. Returns 0 if the
// VIN is too short or the character is invalid.
func modelYearFromVIN(vin string) int {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	if len(vin) < 10 {
		return 0
	}
	base, ok := vinYearCodes[vin[9]]
	if !ok {
		return 0
	}
	cutoff := time.Now().Year() + 1
	best := base
	for y := base + 30; y <= cutoff; y += 30 {
		best = y
	}
	return best
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return strings.Repeat("*", len(t))
	}
	return t[:8] + "…"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
