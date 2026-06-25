package gateway

import "strings"

// EntitlementCategories are UI grouping labels for the entitlement catalog
// (see bell-docs architecture/subscriptions-and-entitlements.md).
var EntitlementCategories = []string{
	"recording",
	"sharing",
	"ai",
	"support",
}

// ValidEntitlementCategory reports whether category is empty or a known label.
func ValidEntitlementCategory(category string) bool {
	c := strings.TrimSpace(category)
	if c == "" {
		return true
	}
	for _, allowed := range EntitlementCategories {
		if c == allowed {
			return true
		}
	}
	return false
}
