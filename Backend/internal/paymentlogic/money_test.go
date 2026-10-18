package paymentlogic

import "testing"

func TestAmountsAndInstallments(t *testing.T) {
	cases := map[string]int64{"8000000": 800000000, "2000000.00": 200000000, "0.01": 1, "125.5": 12550}
	for input, expected := range cases {
		got, err := Cents(input)
		if err != nil || got != expected {
			t.Fatalf("%s: %d %v", input, got, err)
		}
	}
	for _, input := range []string{"-1", "1e3", "1,000", "1.234", "NaN", "", "1.", ".1", "99999999999999999"} {
		if _, err := Cents(input); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	total, _ := Cents("8000000")
	paid, _ := Cents("2000000")
	if Decimal(total-paid) != "6000000.00" {
		t.Fatal("installment balance")
	}
}

func TestGatewayConfirmation(t *testing.T) {
	for _, item := range []struct {
		s, f, c string
		paid    bool
	}{
		{"settlement", "", "200", true}, {"capture", "accept", "200", true},
		{"capture", "challenge", "200", false}, {"pending", "accept", "201", false},
		{"settlement", "deny", "200", false}, {"settlement", "accept", "201", false},
	} {
		if IsPaid(item.s, item.f, item.c) != item.paid {
			t.Fatal(item)
		}
	}
}
