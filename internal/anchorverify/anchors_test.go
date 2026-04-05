package anchorverify

import "testing"

func TestMessageImprintMatches(t *testing.T) {
	ok, err := messageImprintMatches("0a0b0c", oidSHA256, []byte{0x0a, 0x0b, 0x0c})
	if err != nil {
		t.Fatalf("messageImprintMatches returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected digest match")
	}
}

func TestMessageImprintMatches_Mismatch(t *testing.T) {
	ok, err := messageImprintMatches("0a0b0c", oidSHA256, []byte{0x0a, 0x0b, 0x0d})
	if err != nil {
		t.Fatalf("messageImprintMatches returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected digest mismatch")
	}
}
