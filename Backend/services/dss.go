package services

// TentukanPrioritas mengimplementasikan 8 aturan Decision Tree (Akurasi 99.79%)
// berdasarkan Tabel 4.11 dan Lampiran E skripsi Rivansyah Zulfa Nurjaya.
// Fungsi ini menjadi penentu utama jalur distribusi di PT. FAS.
func TentukanPrioritas(jumlah float64, clQty float64) string {
	// Root node: cl_qty <= 18.5
	if clQty <= 18.5 {
		if jumlah <= 605117 {
			return "Klaster 1 (Retail)" // Aturan 1
		}
		// jumlah > 605117
		if clQty <= 12.5 {
			if jumlah <= 1177470 {
				return "Klaster 1 (Retail)" // Aturan 2
			}
			return "Klaster 0 (Reguler)" // Aturan 3
		}
		return "Klaster 0 (Reguler)" // Aturan 4
	}

	// Cabang kanan: cl_qty > 18.5
	if clQty <= 75.5 {
		return "Klaster 0 (Reguler)" // Aturan 5
	}

	// cl_qty > 75.5
	if jumlah <= 1723267 {
		if jumlah <= 1377936 {
			return "Klaster 2 (VVIP)" // Aturan 6
		}
		return "Klaster 0 (Reguler)" // Aturan 7
	}

	return "Klaster 2 (VVIP)" // Aturan 8
}

// RekomendasiDistribusi menerjemahkan label klaster menjadi SOP operasional gudang
// sesuai saran implementasi bertahap pada Bab 5.b.i skripsi.
func RekomendasiDistribusi(cluster string) string {
	switch cluster {
	case "Klaster 1 (Retail)":
		return "Gabungkan rute pengiriman (Milk Run) untuk efisiensi BBM"
	case "Klaster 2 (VVIP)":
		return "PRIORITAS EKSPRES! Gunakan armada dedicated & pastikan Safety Stock 100%"
	default:
		return "Jalur distribusi reguler standar"
	}
}
