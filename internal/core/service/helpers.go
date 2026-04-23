package service

import (
	"strings"

	"controlplane/internal/core/domain/entity"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

var (
	tenantStatuses = map[string]struct{}{
		"active":    {},
		"suspended": {},
		"archived":  {},
	}
	workspaceStatuses = map[string]struct{}{
		"active":   {},
		"disabled": {},
		"archived": {},
	}
)

func normalizePagination(page, limit int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func buildPagination(page, limit int, total int64) entity.Pagination {
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return entity.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

func normalizeGeneratedSlug(value string, maxLen int) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	var builder strings.Builder
	builder.Grow(len(value))

	lastDash := false
	for _, ch := range value {
		isAlphaNum := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if isAlphaNum {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if maxLen > 0 && len(slug) > maxLen {
		slug = strings.Trim(slug[:maxLen], "-")
	}
	return slug
}

func isAllowedStatus(status string, allowed map[string]struct{}) bool {
	_, ok := allowed[status]
	return ok
}
