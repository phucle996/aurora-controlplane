package middleware

import (
	"fmt"
	"testing"
)

func BenchmarkHasPermissionCacheHit(b *testing.B) {
	entry := RoleEntry{
		Level:       1,
		Permissions: make([]string, 0, 256),
	}
	for i := 0; i < 256; i++ {
		entry.Permissions = append(entry.Permissions, fmt.Sprintf("perm:%d", i))
	}
	entry.Permissions = append(entry.Permissions, "smtp:gateway:write")
	entry = normalizeRoleEntry(entry)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !hasPermission(entry, "smtp:gateway:write") {
			b.Fatal("expected permission to be present")
		}
	}
}
