package privacy

import (
	"strings"
	"testing"
)

func TestRedactMasksEmailAndPhone(t *testing.T) {
	in := "клиент ivan@example.com тел +7 (912) 345-67-89 просит счёт"
	out := Redact(in)
	if strings.Contains(out, "ivan@example.com") {
		t.Fatalf("email not masked: %q", out)
	}
	if strings.Contains(out, "345-67-89") {
		t.Fatalf("phone not masked: %q", out)
	}
	if !strings.Contains(out, "[email]") || !strings.Contains(out, "[phone]") {
		t.Fatalf("expected mask tokens: %q", out)
	}
}

func TestRedactLeavesPlainText(t *testing.T) {
	in := "Стерилизация кошки стоит 3500 рублей"
	if out := Redact(in); out != in {
		t.Fatalf("plain text changed: %q", out)
	}
}
