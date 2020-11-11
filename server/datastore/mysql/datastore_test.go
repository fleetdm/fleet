package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
)

func TestAppendListOptionsToSQL(t *testing.T) {
	sql := "SELECT * FROM app_configs"
	opts := kolide.ListOptions{
		OrderKey: "name",
	}

	actual := appendListOptionsToSQL(sql, opts)
	expected := "SELECT * FROM app_configs ORDER BY name ASC LIMIT 1000000"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	sql = "SELECT * FROM app_configs"
	opts.OrderDirection = kolide.OrderDescending
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM app_configs ORDER BY name DESC LIMIT 1000000"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	opts = kolide.ListOptions{
		PerPage: 10,
	}

	sql = "SELECT * FROM app_configs"
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM app_configs LIMIT 10"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	sql = "SELECT * FROM app_configs"
	opts.Page = 2
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM app_configs LIMIT 10 OFFSET 20"
	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

	opts = kolide.ListOptions{}
	sql = "SELECT * FROM app_configs"
	actual = appendListOptionsToSQL(sql, opts)
	expected = "SELECT * FROM app_configs LIMIT 1000000"

	if actual != expected {
		t.Error("Expected", expected, "Actual", actual)
	}

}
