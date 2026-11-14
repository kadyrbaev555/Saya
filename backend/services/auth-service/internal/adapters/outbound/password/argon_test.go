package password

import "testing"

func TestPassword(t *testing.T) {
	p := Argon{}
	a, e := p.Hash("long demo password")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := p.Hash("long demo password")
	if a == b {
		t.Fatal("salt reused")
	}
	if !p.Verify("long demo password", a) || p.Verify("wrong", a) || p.Verify("wrong", "malformed") {
		t.Fatal("verification failed")
	}
}
