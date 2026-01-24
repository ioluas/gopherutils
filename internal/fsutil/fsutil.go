package fsutil

import (
	"os/user"
	"strconv"
	"sync"
	"syscall"
)

var (
	userCache  sync.Map
	groupCache sync.Map
)

// GetOwnerGroup returns the username and group name for a file's UID/GID.
// It uses caching to avoid repeated lookups.
// If lookup fails, it returns the numeric UID/GID as strings.
func GetOwnerGroup(stat *syscall.Stat_t) (string, string) {
	uid := stat.Uid
	gid := stat.Gid

	userStr := resolveUser(uid)
	groupStr := resolveGroup(gid)

	return userStr, groupStr
}

func resolveUser(uid uint32) string {
	uidStr := strconv.FormatUint(uint64(uid), 10)
	if v, ok := userCache.Load(uid); ok {
		return v.(string)
	}

	u, err := user.LookupId(uidStr)
	if err != nil {
		return uidStr
	}
	userCache.Store(uid, u.Username)
	return u.Username
}

func resolveGroup(gid uint32) string {
	gidStr := strconv.FormatUint(uint64(gid), 10)
	if v, ok := groupCache.Load(gid); ok {
		return v.(string)
	}

	g, err := user.LookupGroupId(gidStr)
	if err != nil {
		return gidStr
	}
	groupCache.Store(gid, g.Name)
	return g.Name
}
