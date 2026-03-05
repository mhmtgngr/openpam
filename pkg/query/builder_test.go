package query

import (
	"strings"
	"testing"
)

func TestBuilderBasicSelect(t *testing.T) {
	q, args := From("users").
		Where("tenant_id", "t1").
		Where("status", "active").
		OrderBy("created_at", "desc").
		Paginate(20, 0).
		BuildSelect()

	if !strings.Contains(q, "SELECT * FROM users") {
		t.Errorf("expected SELECT * FROM users, got %s", q)
	}
	if !strings.Contains(q, "tenant_id = $1") {
		t.Errorf("expected tenant_id = $1, got %s", q)
	}
	if !strings.Contains(q, "status = $2") {
		t.Errorf("expected status = $2, got %s", q)
	}
	if !strings.Contains(q, "ORDER BY created_at DESC") {
		t.Errorf("expected ORDER BY, got %s", q)
	}
	if len(args) < 2 {
		t.Errorf("expected at least 2 args, got %d", len(args))
	}
}

func TestBuilderCount(t *testing.T) {
	q, args := From("sessions").
		Where("tenant_id", "t1").
		WhereNull("deleted_at").
		BuildCount()

	if !strings.Contains(q, "SELECT COUNT(*)") {
		t.Errorf("expected COUNT(*), got %s", q)
	}
	if !strings.Contains(q, "deleted_at IS NULL") {
		t.Errorf("expected IS NULL, got %s", q)
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

func TestBuilderWhereLike(t *testing.T) {
	q, args := From("secrets").
		Where("tenant_id", "t1").
		WhereLike("name", "admin").
		BuildSelect()

	if !strings.Contains(q, "ILIKE $2") {
		t.Errorf("expected ILIKE, got %s", q)
	}
	if len(args) < 2 {
		t.Fatalf("expected 2+ args, got %d", len(args))
	}
	if args[1] != "%admin%" {
		t.Errorf("expected %%admin%%, got %v", args[1])
	}
}

func TestBuilderSelectAndCount(t *testing.T) {
	b := From("checkouts").
		Where("tenant_id", "t1").
		Where("status", "active").
		OrderBy("created_at", "desc").
		Paginate(10, 20)

	selectQ, countQ, args := b.BuildSelectAndCount()

	if !strings.Contains(selectQ, "SELECT *") {
		t.Errorf("select query missing SELECT *, got %s", selectQ)
	}
	if !strings.Contains(countQ, "COUNT(*)") {
		t.Errorf("count query missing COUNT(*), got %s", countQ)
	}
	if !strings.Contains(selectQ, "LIMIT") {
		t.Errorf("select query missing LIMIT, got %s", selectQ)
	}
	if strings.Contains(countQ, "LIMIT") {
		t.Errorf("count query should not have LIMIT, got %s", countQ)
	}
	if len(args) < 2 {
		t.Errorf("expected 2+ args, got %d", len(args))
	}
}

func TestBuilderWhereNotNull(t *testing.T) {
	q, _ := From("secrets").
		Where("tenant_id", "t1").
		WhereNotNull("next_rotation_at").
		BuildSelect()

	if !strings.Contains(q, "next_rotation_at IS NOT NULL") {
		t.Errorf("expected IS NOT NULL, got %s", q)
	}
}

func TestBuilderCustomSelect(t *testing.T) {
	q, _ := From("sessions").
		Select("id, user_id, status").
		Where("status", "active").
		BuildSelect()

	if !strings.Contains(q, "SELECT id, user_id, status") {
		t.Errorf("expected custom columns, got %s", q)
	}
}
