package check_app_domains

import (
	"cmp"
	"slices"
)

// By service, then hostname, so the list is stable between checks.
func sortStatuses(statuses []DomainStatus) {
	slices.SortFunc(statuses, func(a, b DomainStatus) int {
		return cmp.Or(cmp.Compare(a.Service, b.Service), cmp.Compare(a.Hostname, b.Hostname))
	})
}
