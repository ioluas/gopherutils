package fsutil

import (
	"os/user"
	"strconv"
	"syscall"
	"testing"
)

func TestGetOwnerGroup(t *testing.T) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	uid, err := strconv.ParseUint(currentUser.Uid, 10, 32)
	if err != nil {
		t.Fatalf("Failed to parse UID: %v", err)
	}

	gid, err := strconv.ParseUint(currentUser.Gid, 10, 32)
	if err != nil {
		t.Fatalf("Failed to parse GID: %v", err)
	}

	stat := &syscall.Stat_t{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}

	// --- First call, should perform lookups ---
	owner, group := GetOwnerGroup(stat)

	// Verify owner
	if owner != currentUser.Username {
		t.Errorf("Expected owner '%s', got '%s'", currentUser.Username, owner)
	}

	// Verify group
	currentGroup, err := user.LookupGroupId(currentUser.Gid)
	if err != nil {
		t.Logf("Skipping group name check, cannot lookup current GID '%s': %v", currentUser.Gid, err)
	} else if group != currentGroup.Name {
		t.Errorf("Expected group '%s', got '%s'", currentGroup.Name, group)
	}

	// --- Second call, should use cache ---
	// To verify caching, we'd ideally mock the user/group lookup functions,
	// but for this test, we'll just ensure the values remain correct.
	cachedOwner, cachedGroup := GetOwnerGroup(stat)

	if cachedOwner != owner {
		t.Errorf("Cached owner '%s' does not match initial owner '%s'", cachedOwner, owner)
	}
	if cachedGroup != group {
		t.Errorf("Cached group '%s' does not match initial group '%s'", cachedGroup, group)
	}
}

func TestGetOwnerGroup_InvalidIds(t *testing.T) {
	// Use a UID/GID that is highly unlikely to exist on any system.
	invalidUID := uint32(999999)
	invalidGID := uint32(999999)

	stat := &syscall.Stat_t{
		Uid: invalidUID,
		Gid: invalidGID,
	}

	owner, group := GetOwnerGroup(stat)

	expectedOwner := strconv.FormatUint(uint64(invalidUID), 10)
	if owner != expectedOwner {
		t.Errorf("Expected owner for invalid UID to be '%s', but got '%s'", expectedOwner, owner)
	}

	expectedGroup := strconv.FormatUint(uint64(invalidGID), 10)
	if group != expectedGroup {
		t.Errorf("Expected group for invalid GID to be '%s', but got '%s'", expectedGroup, group)
	}

	// --- Test caching of invalid IDs ---
	cachedOwner, cachedGroup := GetOwnerGroup(stat)
	if cachedOwner != expectedOwner {
		t.Errorf("Expected cached owner for invalid UID to be '%s', but got '%s'", expectedOwner, cachedOwner)
	}
	if cachedGroup != expectedGroup {
		t.Errorf("Expected cached group for invalid GID to be '%s', but got '%s'", expectedGroup, cachedGroup)
	}
}
