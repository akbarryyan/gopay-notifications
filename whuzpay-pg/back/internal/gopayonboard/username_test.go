package gopayonboard

import "testing"

func TestSanitizeUsername(t *testing.T) {
	cases := map[string]string{
		"budi@toko.com":     "budi",
		"Budi.Santoso@x.co": "budisantoso",
		"a+b@x.co":          "ab",
		"@x.co":             "merchant",
		"":                  "merchant",
		"12345@x.co":        "12345",
	}
	for in, want := range cases {
		if got := SanitizeUsername(in); got != want {
			t.Errorf("SanitizeUsername(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeUsername_DipotongMaks20Karakter(t *testing.T) {
	got := SanitizeUsername("initigapanjangbangetlebihdari20karakter@x.co")
	if len(got) > 20 {
		t.Errorf("panjang = %d, mau maks 20", len(got))
	}
}
