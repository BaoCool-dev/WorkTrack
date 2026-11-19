package securetoken_test

import (
	"bytes"
	"testing"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/securetoken"
)

func TestNew_UniqueAndHashMatches_B7(t *testing.T) {
	a, ha, err := securetoken.New()
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := securetoken.New()
	if a == b {
		t.Fatal("hai token trùng nhau")
	}
	if len(a) != 43 {
		t.Fatalf("độ dài base64url 32 byte phải là 43, got %d", len(a))
	}
	if !bytes.Equal(ha, securetoken.Hash(a)) || len(ha) != 32 {
		t.Fatal("hash không khớp")
	}
}
