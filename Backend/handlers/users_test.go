package handlers

import (
	"encoding/json"
	"furi-backend/middleware"
	"furi-backend/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagedUserValidation(t *testing.T) {
	valid := userChange{Name: " Finance Uji ", Email: " STAFF@EXAMPLE.COM ", Role: "finance", IsActive: true, Password: "test-only-password"}
	if msg := validUserChange(&valid, false, true); msg != "" {
		t.Fatal(msg)
	}
	if valid.Email != "staff@example.com" || valid.Name != "Finance Uji" {
		t.Fatal("normalization failed")
	}
	for _, role := range []string{"owner", "root", ""} {
		x := valid
		x.Role = role
		if validUserChange(&x, false, true) == "" {
			t.Fatal("accepted role", role)
		}
	}
	for _, email := range []string{"invalid", "User <a@example.com>", "a@example.com,b@example.com"} {
		x := valid
		x.Email = email
		if validUserChange(&x, false, true) == "" {
			t.Fatal("accepted email", email)
		}
	}
	for _, password := range []string{"short", strings.Repeat("x", 73), strings.Repeat("漢", 25)} {
		x := valid
		x.Password = password
		if validUserChange(&x, false, true) == "" {
			t.Fatal("accepted password size")
		}
	}
	x := valid
	x.Password = ""
	if validUserChange(&x, false, false) != "" {
		t.Fatal("edit should not require password")
	}
	x.Password = "new-password"
	if validUserChange(&x, false, false) == "" {
		t.Fatal("edit must not silently change password")
	}
}
func TestManagedUserNeverSerializesSecrets(t *testing.T) {
	data, e := json.Marshal(managedUser{ID: "id", Name: "Uji", Email: "qa@example.test", Role: "finance", Revision: 2})
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(data), "password") {
		t.Fatal("secret field exposed")
	}
}
func TestUsersRoleGate(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-only-secret-not-for-deployment")
	called := false
	endpoint := middleware.Auth(middleware.RequireRole(func(w http.ResponseWriter, r *http.Request) { called = true }, "admin"))
	token, e := services.GenerateJWT("00000000-0000-4000-8000-000000000001", "qa@example.test", "finance", 3)
	if e != nil {
		t.Fatal(e)
	}
	claims, e := services.ValidateJWT(token)
	if e != nil || claims.AuthRevision != 3 {
		t.Fatal("revision claim missing", e)
	}
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	endpoint(w, req)
	if w.Code != 403 || called {
		t.Fatal("non-admin reached handler")
	}
	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	w = httptest.NewRecorder()
	endpoint(w, req)
	if w.Code != 401 {
		t.Fatal("missing auth accepted")
	}
}
