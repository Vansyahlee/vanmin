package expenselogic

import (
	"encoding/json"
	"errors"
	"furi-backend/internal/paymentlogic"
	"strings"
	"time"
)

var ErrDetails = errors.New("rincian gaji atau supplier tidak valid; periksa periode, referensi invoice, dan perhitungan nominal")

const MaxCents int64 = 9999999999999999

type Details struct {
	EmployeeCode      string `json:"employee_code,omitempty"`
	PayrollPeriod     string `json:"payroll_period,omitempty"`
	BasicSalary       string `json:"basic_salary,omitempty"`
	Allowance         string `json:"allowance,omitempty"`
	Deduction         string `json:"deduction,omitempty"`
	SupplierInvoice   string `json:"supplier_invoice_number,omitempty"`
	SupplierDate      string `json:"supplier_invoice_date,omitempty"`
	PurchaseReference string `json:"purchase_reference,omitempty"`
}

func Category(kind, current string) string {
	if kind == "SALARY" {
		return "Gaji & Honor"
	}
	if kind == "SUPPLIER" {
		return "Pembayaran Supplier"
	}
	return current
}
func Normalize(kind string, d Details, amount string) (json.RawMessage, error) {
	if kind == "" {
		kind = "GENERAL"
	}
	d.EmployeeCode = strings.TrimSpace(d.EmployeeCode)
	d.PayrollPeriod = strings.TrimSpace(d.PayrollPeriod)
	d.SupplierInvoice = strings.TrimSpace(d.SupplierInvoice)
	d.SupplierDate = strings.TrimSpace(d.SupplierDate)
	d.PurchaseReference = strings.TrimSpace(d.PurchaseReference)
	total, err := paymentlogic.Cents(amount)
	if err != nil || total <= 0 || total > MaxCents {
		return nil, ErrDetails
	}
	if len(d.EmployeeCode) > 80 || len(d.SupplierInvoice) > 150 || len(d.PurchaseReference) > 150 {
		return nil, ErrDetails
	}
	switch kind {
	case "GENERAL":
		if d != (Details{}) {
			return nil, ErrDetails
		}
	case "SALARY":
		if d.SupplierInvoice != "" || d.SupplierDate != "" || d.PurchaseReference != "" {
			return nil, ErrDetails
		}
		period, e := time.Parse("2006-01", d.PayrollPeriod)
		if e != nil || period.Year() < 2000 {
			return nil, ErrDetails
		}
		basic, e1 := paymentlogic.Cents(d.BasicSalary)
		if d.Allowance == "" {
			d.Allowance = "0"
		}
		if d.Deduction == "" {
			d.Deduction = "0"
		}
		allowance, e2 := paymentlogic.Cents(d.Allowance)
		deduction, e3 := paymentlogic.Cents(d.Deduction)
		if e1 != nil || e2 != nil || e3 != nil || basic > MaxCents || allowance > MaxCents || deduction > MaxCents || basic+allowance > MaxCents || deduction >= basic+allowance || basic+allowance-deduction != total {
			return nil, ErrDetails
		}
		d.BasicSalary = paymentlogic.Decimal(basic)
		d.Allowance = paymentlogic.Decimal(allowance)
		d.Deduction = paymentlogic.Decimal(deduction)
	case "SUPPLIER":
		if d.EmployeeCode != "" || d.PayrollPeriod != "" || d.BasicSalary != "" || d.Allowance != "" || d.Deduction != "" || d.SupplierInvoice == "" {
			return nil, ErrDetails
		}
		if d.SupplierDate != "" {
			date, e := time.Parse("2006-01-02", d.SupplierDate)
			if e != nil || date.Year() < 2000 {
				return nil, ErrDetails
			}
		}
	default:
		return nil, ErrDetails
	}
	return json.Marshal(d)
}
