package gateway

import "strings"

// EntitlementUnits are suffix labels for number-type entitlements
// (see bell-docs architecture/subscriptions-and-entitlements.md).
var EntitlementUnits = []string{
	"days",
	"seconds",
	"count",
	"kbps",
}

// ValidEntitlementUnit reports whether unit is empty or a known label.
func ValidEntitlementUnit(unit string) bool {
	u := strings.TrimSpace(unit)
	if u == "" {
		return true
	}
	for _, allowed := range EntitlementUnits {
		if u == allowed {
			return true
		}
	}
	return false
}
