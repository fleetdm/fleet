package common_mysql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DefaultSelectLimit is used if no limit is provided in ListOptions.
const DefaultSelectLimit = 1000000

// columnCharsRegexp matches characters not allowed in column names.
var columnCharsRegexp = regexp.MustCompile(`[^\w-.]`)

// SanitizeColumn sanitizes a column name for safe use in SQL queries.
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

// OrderDirection defines the order direction for sorting.
type OrderDirection int

const (
	// OrderAscending sorts in ascending order.
	OrderAscending OrderDirection = iota
	// OrderDescending sorts in descending order.
	OrderDescending
)

// ListOptions defines pagination and sorting options for SQL queries.
// This is a generic version that can be used by any bounded context.
type ListOptions struct {
	Page                        uint
	PerPage                     uint
	OrderKey                    string
	OrderDirection              OrderDirection
	MatchQuery                  string
	After                       string
	IncludeMetadata             bool
	TestSecondaryOrderKey       string
	TestSecondaryOrderDirection OrderDirection
}

// AppendListOptionsWithCursorToSQL appends ORDER BY, LIMIT, and OFFSET clauses
// to the SQL query based on the provided ListOptions.
func AppendListOptionsWithCursorToSQL(sql string, params []any, opts *ListOptions) (string, []any) {
	orderKey := SanitizeColumn(opts.OrderKey)

	if opts.After != "" && orderKey != "" {
		afterSQL := " WHERE "
		if strings.Contains(strings.ToLower(sql), "where") {
			afterSQL = " AND "
		}
		if strings.HasSuffix(orderKey, "id") {
			i, _ := strconv.Atoi(opts.After)
			params = append(params, i)
		} else {
			params = append(params, opts.After)
		}
		direction := ">" // ASC
		if opts.OrderDirection == OrderDescending {
			direction = "<" // DESC
		}
		sql = fmt.Sprintf("%s %s %s %s ?", sql, afterSQL, orderKey, direction)

		// After existing supersedes Page, so we disable it
		opts.Page = 0
	}

	if orderKey != "" {
		direction := "ASC"
		if opts.OrderDirection == OrderDescending {
			direction = "DESC"
		}

		sql = fmt.Sprintf("%s ORDER BY %s %s", sql, orderKey, direction)
		if opts.TestSecondaryOrderKey != "" {
			direction := "ASC"
			if opts.TestSecondaryOrderDirection == OrderDescending {
				direction = "DESC"
			}
			sql += fmt.Sprintf(`, %s %s`, SanitizeColumn(opts.TestSecondaryOrderKey), direction)
		}
	}
	// If caller doesn't supply a limit, apply a default limit to ensure
	// that an unbounded query with many results doesn't consume too much memory.
	if opts.PerPage == 0 {
		opts.PerPage = DefaultSelectLimit
	}

	perPage := opts.PerPage
	if opts.IncludeMetadata {
		perPage++
	}
	sql = fmt.Sprintf("%s LIMIT %d", sql, perPage)

	offset := opts.PerPage * opts.Page

	if offset > 0 {
		sql = fmt.Sprintf("%s OFFSET %d", sql, offset)
	}

	return sql, params
}
