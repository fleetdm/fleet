package mysql

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// columnCharsRegexp matches characters that are not allowed in column names.
var columnCharsRegexp = regexp.MustCompile(`[^\w-.]`)

// OrderKeyAllowlist maps user-facing order key names to actual SQL column expressions.
// For example: {"hostname": "h.hostname", "created_at": "h.created_at"}
// An empty map means no sorting is allowed.
// A nil map will cause a panic to catch programming errors during development.
type OrderKeyAllowlist map[string]string

// InvalidOrderKeyError is returned when an order_key is not in the allowlist.
type InvalidOrderKeyError struct {
	Key     string
	Allowed []string
}

func (e InvalidOrderKeyError) Error() string {
	return fmt.Sprintf("invalid order_key: %q; allowed values are: %v", e.Key, e.Allowed)
}

// AllowedKeys returns the sorted list of allowed keys for error messages.
func (a OrderKeyAllowlist) AllowedKeys() []string {
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ListOptions defines the interface for pagination and sorting options.
// This interface allows the common_mysql package to work with various list options implementations.
type ListOptions interface {
	GetPage() uint
	GetPerPage() uint
	GetOrderKey() string
	IsDescending() bool
	GetCursorValue() string
	WantsPaginationInfo() bool
	GetSecondaryOrderKey() string
	IsSecondaryDescending() bool
}

// SanitizeColumn sanitizes a column name for use in SQL queries.
// It removes invalid characters and wraps parts in backticks.
func SanitizeColumn(col string) string {
	col = columnCharsRegexp.ReplaceAllString(col, "")
	oldParts := strings.Split(col, ".")
	parts := oldParts[:0]
	for _, p := range oldParts {
		if len(p) != 0 {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	col = "`" + strings.Join(parts, "`.`") + "`"
	return col
}

// AppendListOptionsWithParamsSecure appends ORDER BY, LIMIT, and OFFSET clauses to a SQL string
// based on the provided list options. It validates the order key against the provided
// allowlist to prevent SQL injection and information disclosure via arbitrary column sorting.
//
// The allowlist maps user-facing key names to actual SQL column expressions.
// If the order key is not in the allowlist, an InvalidOrderKeyError is returned.
// If the order key is empty, no ORDER BY clause is added (no error).
// If allowlist is nil, the function will panic (programming error).
// If allowlist is empty, any non-empty order key will return an error.
func AppendListOptionsWithParamsSecure(sql string, params []any, opts ListOptions, allowlist OrderKeyAllowlist) (string, []any, error) {
	if allowlist == nil {
		panic("AppendListOptionsWithParams: allowlist cannot be nil; use empty map to disallow all sorting")
	}

	userOrderKey := opts.GetOrderKey()
	var orderKey string

	// Validate and translate order key
	if userOrderKey != "" {
		actualColumn, ok := allowlist[userOrderKey]
		if !ok {
			return "", nil, InvalidOrderKeyError{
				Key:     userOrderKey,
				Allowed: allowlist.AllowedKeys(),
			}
		}

		orderKey = actualColumn
	}

	page := opts.GetPage()

	if cursor := opts.GetCursorValue(); cursor != "" && orderKey != "" {
		cursorSQL := " WHERE "
		if strings.Contains(strings.ToLower(sql), "where") {
			cursorSQL = " AND "
		}
		// Cursor value is always passed as string. MySQL automatically converts
		// string to integer when comparing against integer columns.
		// See: https://dev.mysql.com/doc/refman/8.0/en/type-conversion.html
		params = append(params, cursor)
		direction := ">" // ASC
		if opts.IsDescending() {
			direction = "<" // DESC
		}
		sql = fmt.Sprintf("%s %s %s %s ?", sql, cursorSQL, orderKey, direction)

		// Cursor-based pagination supersedes page-based pagination
		page = 0
	}

	if orderKey != "" {
		direction := "ASC"
		if opts.IsDescending() {
			direction = "DESC"
		}

		sql = fmt.Sprintf("%s ORDER BY %s %s", sql, orderKey, direction)

		// Handle secondary order key (used for test determinism)
		if secondaryKey := opts.GetSecondaryOrderKey(); secondaryKey != "" {
			// Secondary key must also be in allowlist
			if actualSecondary, ok := allowlist[secondaryKey]; ok {
				dir := "ASC"
				if opts.IsSecondaryDescending() {
					dir = "DESC"
				}
				sql += fmt.Sprintf(`, %s %s`, actualSecondary, dir)
			}
			// If secondary key not in allowlist, silently ignore (it's optional/for tests)
		}
	}

	limit := opts.GetPerPage()
	if opts.WantsPaginationInfo() {
		limit++
	}
	sql = fmt.Sprintf("%s LIMIT %d", sql, limit)

	offset := opts.GetPerPage() * page
	if offset > 0 {
		sql = fmt.Sprintf("%s OFFSET %d", sql, offset)
	}

	return sql, params, nil
}

// AppendListOptionsWithParams appends ORDER BY, LIMIT, and OFFSET clauses to a SQL string
// based on the provided list options. It accepts existing query params and returns
// the extended params slice.
//
// Deprecated: this method will be removed in favor of AppendListOptionsWithParamsSecure
func AppendListOptionsWithParams(sql string, params []any, opts ListOptions) (string, []any) {
	orderKey := SanitizeColumn(opts.GetOrderKey())
	page := opts.GetPage()

	if cursor := opts.GetCursorValue(); cursor != "" && orderKey != "" {
		cursorSQL := " WHERE "
		if strings.Contains(strings.ToLower(sql), "where") {
			cursorSQL = " AND "
		}
		// Cursor value is always passed as string. MySQL automatically converts
		// string to integer when comparing against integer columns.
		// See: https://dev.mysql.com/doc/refman/8.0/en/type-conversion.html
		params = append(params, cursor)
		direction := ">" // ASC
		if opts.IsDescending() {
			direction = "<" // DESC
		}
		sql = fmt.Sprintf("%s %s %s %s ?", sql, cursorSQL, orderKey, direction)

		// Cursor-based pagination supersedes page-based pagination
		page = 0
	}

	if orderKey != "" {
		direction := "ASC"
		if opts.IsDescending() {
			direction = "DESC"
		}

		sql = fmt.Sprintf("%s ORDER BY %s %s", sql, orderKey, direction)
		if opts.GetSecondaryOrderKey() != "" {
			dir := "ASC"
			if opts.IsSecondaryDescending() {
				dir = "DESC"
			}
			sql += fmt.Sprintf(`, %s %s`, SanitizeColumn(opts.GetSecondaryOrderKey()), dir)
		}
	}

	limit := opts.GetPerPage()
	if opts.WantsPaginationInfo() {
		limit++
	}
	sql = fmt.Sprintf("%s LIMIT %d", sql, limit)

	offset := opts.GetPerPage() * page
	if offset > 0 {
		sql = fmt.Sprintf("%s OFFSET %d", sql, offset)
	}

	return sql, params
}
