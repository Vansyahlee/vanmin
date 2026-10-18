package expenselogic

import (
	"encoding/json"
	"testing"
)

func TestSalaryNetMatchesActualExpense(t *testing.T) {
	d := Details{PayrollPeriod: "2026-10", BasicSalary: "4000000", Allowance: "500000.25", Deduction: "200000.10"}
	raw, err := Normalize("SALARY", d, "4300000.15")
	if err != nil {
		t.Fatal(err)
	}
	var normalized Details
	if json.Unmarshal(raw, &normalized) != nil || normalized.BasicSalary != "4000000.00" {
		t.Fatal(string(raw))
	}
	for _, amount := range []string{"4300000.14", "4500000.25", "0"} {
		if _, err := Normalize("SALARY", d, amount); err == nil {
			t.Fatal("incorrect net accepted", amount)
		}
	}
	d.Deduction = "4500000.25"
	if _, err := Normalize("SALARY", d, "1"); err == nil {
		t.Fatal("deduction consumes salary")
	}
}
func TestExpenseKindDetailsAreSeparated(t *testing.T) {
	if _, err := Normalize("GENERAL", Details{}, "100000"); err != nil {
		t.Fatal(err)
	}
	if _, err := Normalize("SUPPLIER", Details{SupplierInvoice: "INV-PABRIK-01", SupplierDate: "2026-10-01"}, "100000"); err != nil {
		t.Fatal(err)
	}
	for _, d := range []Details{{}, {SupplierInvoice: "X", PayrollPeriod: "2026-10"}, {SupplierInvoice: "X", SupplierDate: "2026-02-30"}} {
		if _, err := Normalize("SUPPLIER", d, "100000"); err == nil {
			t.Fatal("invalid supplier details", d)
		}
	}
	if _, err := Normalize("GENERAL", Details{SupplierInvoice: "X"}, "100000"); err == nil {
		t.Fatal("unclassified supplier details accepted")
	}
	if _, err := Normalize("UNSUPPORTED", Details{}, "100000"); err == nil {
		t.Fatal("unknown kind")
	}
}
