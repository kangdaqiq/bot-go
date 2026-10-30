package services

import (
	"reflect"
	"sort"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"08123456789", "628123456789"},
		{"8123456789", "628123456789"},
		{"628123456789", "628123456789"},
		{"+628123456789", "628123456789"},
		{"+62 812-3456-7890", "6281234567890"},
		{"60123456789", "60123456789"},           // Malaysia
		{"+60 12-345 6789", "60123456789"},        // Malaysia with + and spaces
		{"0060123456789", "60123456789"},          // Malaysia with 00 prefix
		{"966512345678", "966512345678"},          // Saudi Arabia
		{"+1 (415) 555-2671", "14155552671"},       // USA
	}

	for _, tt := range tests {
		actual := NormalizePhone(tt.input)
		if actual != tt.expected {
			t.Errorf("NormalizePhone(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestPhoneVariants(t *testing.T) {
	variants := phoneVariants("+60 12-345 6789")
	sort.Strings(variants)
	expected := []string{"+60123456789", "0060123456789", "60123456789"}
	sort.Strings(expected)
	if !reflect.DeepEqual(variants, expected) {
		t.Errorf("phoneVariants(+60 12-345 6789) = %v, expected %v", variants, expected)
	}

	idVariants := phoneVariants("08123456789")
	has62 := false
	has0 := false
	for _, v := range idVariants {
		if v == "628123456789" {
			has62 = true
		}
		if v == "08123456789" {
			has0 = true
		}
	}
	if !has62 || !has0 {
		t.Errorf("phoneVariants(08123456789) missing expected variants: %v", idVariants)
	}
}
