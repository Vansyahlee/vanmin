package paymentlogic

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrAmount = errors.New("nominal harus berupa angka positif, maksimal 2 desimal, tanpa pemisah ribuan")

// Decimal strings avoid rounding money via float64.
func Cents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 || len(parts[0]) > 16 {
		return 0, ErrAmount
	}
	for _, c := range value {
		if (c < '0' || c > '9') && c != '.' {
			return 0, ErrAmount
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrAmount
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, ErrAmount
		}
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	cents, _ := strconv.ParseInt(fraction, 10, 64)
	return whole*100 + cents, nil
}

func Decimal(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

func IsPaid(status, fraud, code string) bool {
	return code == "200" && (status == "settlement" || (status == "capture" && fraud == "accept")) && (fraud == "" || fraud == "accept")
}
