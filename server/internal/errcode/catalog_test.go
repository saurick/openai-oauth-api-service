package errcode

import "testing"

func TestDefinitionsUseUniqueCodes(t *testing.T) {
	seen := make(map[int32]string, len(Definitions()))
	for _, item := range Definitions() {
		if prev, ok := seen[item.Code]; ok {
			t.Fatalf("duplicate code=%d: %s vs %s", item.Code, prev, item.Name)
		}
		seen[item.Code] = item.Name
	}
}

func TestIsAuthFailureCode(t *testing.T) {
	if !IsAuthFailureCode(AuthExpired.Code) {
		t.Fatalf("auth expired should require relogin")
	}
	if !IsAuthFailureCode(AuthInvalid.Code) {
		t.Fatalf("auth invalid should require relogin")
	}
	if !IsAuthFailureCode(AuthRequired.Code) {
		t.Fatalf("auth required should require relogin")
	}
	if IsAuthFailureCode(PermissionDenied.Code) {
		t.Fatalf("permission denied must not be treated as relogin")
	}
}
