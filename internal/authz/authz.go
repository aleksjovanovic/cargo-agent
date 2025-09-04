// internal/authz/authz.go
package authz

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	adminOnce sync.Once
	adminIDs  map[int64]struct{}
)

func loadAdminIDs() {
	adminIDs = make(map[int64]struct{})
	csv := strings.TrimSpace(os.Getenv("ADMIN_USER_IDS")) // for example "1,2,42"
	if csv == "" {
		return
	}
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.ParseInt(p, 10, 64); err == nil && n > 0 {
			adminIDs[n] = struct{}{}
		}
	}
}

// IsAdmin returns true if userID is in ADMIN_USER_IDS (comma-separated ints).
func IsAdmin(userID int64) bool {
	adminOnce.Do(loadAdminIDs)
	_, ok := adminIDs[userID]
	return ok
}
