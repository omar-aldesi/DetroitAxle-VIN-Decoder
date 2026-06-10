package services

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// --- Brand detection ---

// IsGMBrandVIN reports whether the VIN's World Manufacturer Identifier belongs
// to a standard GM plant. NOTE: this is a fast-path hint only — some GM-branded
// vehicles (e.g. upfitter-built chassis cabs) carry a non-GM WMI, so callers
// should not hard-reject a VIN solely because this returns false.
//
//   - GM North America (USA / Canada / Mexico): WMI starts with 1G, 2G, 3G
//   - GM Korea (Chevrolet/Buick/Cadillac built in Korea): KL4, KL8, KL1
//   - GM Germany (Opel): W0L
func IsGMBrandVIN(vin string) bool {
	if len(vin) < 3 {
		return false
	}
	upper := strings.ToUpper(vin)
	switch upper[:2] {
	case "1G", "2G", "3G":
		return true
	}
	switch upper[:3] {
	case "KL4", "KL8", "KL1", "W0L":
		return true
	}
	return false
}

// --- Shared utilities ---

// gmUUID generates a random UUID v4 string using crypto/rand (no external deps).
func gmUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(t >> (i % 8 * 8))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
