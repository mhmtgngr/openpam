package query

import (
	"fmt"
	"strings"
)

// Builder constructs SQL queries with parameterized conditions
// This replaces manual string concatenation + argCount tracking throughout the codebase
type Builder struct {
	table      string
	selects    string
	conditions []condition
	orderBy    string
	limit      int
	offset     int
	args       []interface{}
	argIdx     int
}

type condition struct {
	clause string
	args   []interface{}
}

// From creates a new query builder for the given table
func From(table string) *Builder {
	return &Builder{
		table:   table,
		selects: "*",
		argIdx:  1,
	}
}

// Select specifies columns to select
func (b *Builder) Select(columns string) *Builder {
	b.selects = columns
	return b
}

// Where adds a condition with a parameterized value
func (b *Builder) Where(column string, value interface{}) *Builder {
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s = $%d", column, b.argIdx),
		args:   []interface{}{value},
	})
	b.argIdx++
	return b
}

// WhereNotNull adds an IS NOT NULL condition
func (b *Builder) WhereNotNull(column string) *Builder {
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IS NOT NULL", column),
	})
	return b
}

// WhereNull adds an IS NULL condition
func (b *Builder) WhereNull(column string) *Builder {
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IS NULL", column),
	})
	return b
}

// WhereIn adds an IN condition
func (b *Builder) WhereIn(column string, values []interface{}) *Builder {
	if len(values) == 0 {
		return b
	}

	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = fmt.Sprintf("$%d", b.argIdx)
		b.argIdx++
		b.args = append(b.args, v)
	}

	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")),
	})
	return b
}

// WhereLike adds a LIKE condition (case-insensitive via ILIKE)
func (b *Builder) WhereLike(column string, pattern string) *Builder {
	b.conditions = append(b.conditions, condition{
		clause: fmt.Sprintf("%s ILIKE $%d", column, b.argIdx),
		args:   []interface{}{"%" + pattern + "%"},
	})
	b.argIdx++
	return b
}

// WhereRaw adds a raw SQL condition (for complex expressions)
func (b *Builder) WhereRaw(sql string, args ...interface{}) *Builder {
	// Replace ? placeholders with $N
	replaced := sql
	for _, arg := range args {
		replaced = strings.Replace(replaced, "?", fmt.Sprintf("$%d", b.argIdx), 1)
		b.argIdx++
		_ = arg
	}
	b.conditions = append(b.conditions, condition{
		clause: replaced,
		args:   args,
	})
	return b
}

// WhereOptional adds a condition only if the value is non-nil
func WhereOptional[T any](b *Builder, column string, value *T) *Builder {
	if value == nil {
		return b
	}
	return b.Where(column, *value)
}

// OrderBy sets the ORDER BY clause
func (b *Builder) OrderBy(column string, direction string) *Builder {
	if direction != "asc" && direction != "desc" {
		direction = "desc"
	}
	b.orderBy = fmt.Sprintf("%s %s", column, strings.ToUpper(direction))
	return b
}

// Paginate sets LIMIT and OFFSET
func (b *Builder) Paginate(limit, offset int) *Builder {
	b.limit = limit
	b.offset = offset
	return b
}

// BuildSelect builds a SELECT query
func (b *Builder) BuildSelect() (string, []interface{}) {
	query := fmt.Sprintf("SELECT %s FROM %s", b.selects, b.table)
	args := b.collectArgs()

	if len(b.conditions) > 0 {
		clauses := make([]string, len(b.conditions))
		for i, c := range b.conditions {
			clauses[i] = c.clause
		}
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	if b.orderBy != "" {
		query += " ORDER BY " + b.orderBy
	}

	if b.limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", b.argIdx)
		args = append(args, b.limit)
		b.argIdx++
	}

	if b.offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", b.argIdx)
		args = append(args, b.offset)
		b.argIdx++
	}

	return query, args
}

// BuildCount builds a COUNT query with the same conditions
func (b *Builder) BuildCount() (string, []interface{}) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", b.table)
	args := b.collectArgs()

	if len(b.conditions) > 0 {
		clauses := make([]string, len(b.conditions))
		for i, c := range b.conditions {
			clauses[i] = c.clause
		}
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	return query, args
}

func (b *Builder) collectArgs() []interface{} {
	var args []interface{}
	for _, c := range b.conditions {
		args = append(args, c.args...)
	}
	return args
}

// BuildSelectAndCount returns both SELECT and COUNT queries sharing the same WHERE conditions
func (b *Builder) BuildSelectAndCount() (selectQuery string, countQuery string, args []interface{}) {
	args = b.collectArgs()

	whereClause := ""
	if len(b.conditions) > 0 {
		clauses := make([]string, len(b.conditions))
		for i, c := range b.conditions {
			clauses[i] = c.clause
		}
		whereClause = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s%s", b.table, whereClause)
	selectQuery = fmt.Sprintf("SELECT %s FROM %s%s", b.selects, b.table, whereClause)

	if b.orderBy != "" {
		selectQuery += " ORDER BY " + b.orderBy
	}

	if b.limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT $%d", b.argIdx)
		args = append(args, b.limit)
		b.argIdx++
	}

	if b.offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET $%d", b.argIdx)
		args = append(args, b.offset)
		b.argIdx++
	}

	return selectQuery, countQuery, args
}
