package mysql

import (
	"fmt"
	"regexp"
	"strings"
)

// columnCharsRegexp matches characters that are not allowed in column names.
var columnCharsRegexp = regexp.MustCompile(`[^\w-.]`)

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

// AppendListOptions appends ORDER BY, LIMIT, and OFFSET clauses to a SQL string
// based on the provided list options.
func AppendListOptions(sql string, opts ListOptions) (string, []any) {
	return AppendListOptionsWithParams(sql, nil, opts)
}

// AppendListOptionsWithParams appends ORDER BY, LIMIT, and OFFSET clauses to a SQL string
// based on the provided list options. It accepts existing query params and returns
// the extended params slice.
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
