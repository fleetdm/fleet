package mcp_servers

import (
	"context"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
)

type mockClient struct {
	row  map[string]string
	rows []map[string]string
	err  error
}

func (m *mockClient) QueryRowContext(ctx context.Context, sql string) (map[string]string, error) {
	return m.row, m.err
}

func (m *mockClient) QueryRowsContext(ctx context.Context, sql string) ([]map[string]string, error) {
	return m.rows, m.err
}
func (m *mockClient) Close() {}

func TestBuildSQL_Unconstrained(t *testing.T) {
	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}
	sql, single, err := buildSQL(qc)
	if err != nil {
		t.Fatal(err)
	}
	if single {
		t.Fatalf("expected single=false")
	}
	if want := "SELECT pid, name, cmdline FROM processes LIMIT 5000"; sql != want {
		t.Fatalf("sql mismatch: %q", sql)
	}
}

func TestBuildSQL_PidEquals(t *testing.T) {
	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{
		"pid": {Constraints: []table.Constraint{{Expression: "123", Operator: table.OperatorEquals}}},
	}}
	sql, single, err := buildSQL(qc)
	if err != nil {
		t.Fatal(err)
	}
	if !single {
		t.Fatalf("expected single=true")
	}
	if want := "SELECT pid, name, cmdline FROM processes WHERE pid = 123"; sql != want {
		t.Fatalf("sql mismatch: %q", sql)
	}
}

func TestBuildSQL_NameLike(t *testing.T) {
	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{
		"name": {Constraints: []table.Constraint{{Expression: "ssh%", Operator: table.OperatorLike}}},
	}}
	sql, single, err := buildSQL(qc)
	if err != nil {
		t.Fatal(err)
	}
	if single {
		t.Fatalf("expected single=false")
	}
	if want := "SELECT pid, name, cmdline FROM processes WHERE (name LIKE 'ssh%')"; sql != want {
		t.Fatalf("sql mismatch: %q", sql)
	}
}

func TestGenerate_MultiRows(t *testing.T) {
	old := newClient
	defer func() { newClient = old }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{rows: []map[string]string{{"pid": "1", "name": "a", "cmdline": "a"}, {"pid": "2", "name": "b", "cmdline": "b"}}}, nil
	}

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{
		"name": {Constraints: []table.Constraint{{Expression: "a%", Operator: table.OperatorLike}}},
	}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestGenerate_SingleRow(t *testing.T) {
	old := newClient
	defer func() { newClient = old }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{row: map[string]string{"pid": "123", "name": "proc", "cmdline": "/proc"}}, nil
	}

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{
		"pid": {Constraints: []table.Constraint{{Expression: "123", Operator: table.OperatorEquals}}},
	}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}
