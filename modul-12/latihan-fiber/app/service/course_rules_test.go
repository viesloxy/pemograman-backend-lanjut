package service

import "testing"

// Table-driven test: setiap baris tabel adalah satu skenario. Menambah
// skenario cukup dengan menambah satu baris, tanpa menyalin kode assertion.
func TestWithinCreditLimit(t *testing.T) {
	tests := []struct {
		name   string
		taken  int
		adding int
		want   bool
	}{
		{name: "KRS masih kosong", taken: 0, adding: 3, want: true},
		{name: "masih jauh dari batas", taken: 12, adding: 3, want: true},
		{name: "tepat di batas", taken: 21, adding: 3, want: true},
		{name: "melewati batas satu SKS", taken: 22, adding: 3, want: false},
		{name: "KRS sudah penuh", taken: 24, adding: 1, want: false},
	}

	for _, tt := range tests {
		// t.Run membuat subtest. Bila satu baris gagal, baris lain tetap
		// dijalankan dan namanya muncul di output, sehingga kita langsung
		// tahu skenario mana yang rusak.
		t.Run(tt.name, func(t *testing.T) {
			got := WithinCreditLimit(tt.taken, tt.adding)
			if got != tt.want {
				t.Errorf("WithinCreditLimit(%d, %d) = %v, seharusnya %v",
					tt.taken, tt.adding, got, tt.want)
			}
		})
	}
}

func TestCreditsAfterSwap(t *testing.T) {
	tests := []struct {
		name                   string
		taken, leaving, joining int
		want                   int
	}{
		{name: "pindah ke kelas dengan SKS sama", taken: 18, leaving: 3, joining: 3, want: 18},
		{name: "pindah ke kelas yang lebih berat", taken: 18, leaving: 2, joining: 4, want: 20},
		{name: "pindah ke kelas yang lebih ringan", taken: 18, leaving: 4, joining: 2, want: 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreditsAfterSwap(tt.taken, tt.leaving, tt.joining)
			if got != tt.want {
				t.Errorf("CreditsAfterSwap(%d, %d, %d) = %d, seharusnya %d",
					tt.taken, tt.leaving, tt.joining, got, tt.want)
			}
		})
	}
}