package utils

import "strconv"

func ParsePagination(limitRaw, offsetRaw string, defaultLimit, maxLimit int) (limit int, offset int) {
	limit = defaultLimit
	offset = 0

	if parsed, err := strconv.Atoi(limitRaw); err == nil && parsed > 0 {
		limit = parsed
	}
	if parsed, err := strconv.Atoi(offsetRaw); err == nil && parsed >= 0 {
		offset = parsed
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, offset
}
