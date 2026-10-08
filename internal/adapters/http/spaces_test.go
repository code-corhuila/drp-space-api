package httpadapter

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/code-corhuila/drp-space-api/internal/adapters/memory"
	"github.com/code-corhuila/drp-space-api/internal/adapters/security"
	"github.com/code-corhuila/drp-space-api/internal/app"
	"github.com/golang-jwt/jwt/v5"
)

const testKID = "spacehub-identity-2026"

type testIssuer struct {
	priv *rsa.PrivateKey
	ver  *security.Verifier
}

func spaceMux(t *testing.T) (http.Handler, testIssuer) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := security.PublicJWKS(&priv.PublicKey, testKID)
	ver := security.NewVerifier(security.StaticJWKS(jwks))
	spaces, err := memory.Corte2()
	if err != nil {
		t.Fatal(err)
	}
	return NewMux(Deps{
		Service: "space-service",
		Catalog: app.Catalog{Spaces: spaces},
		Tokens:  ver,
	}), testIssuer{priv: priv, ver: ver}
}

func (i testIssuer) bearer(t *testing.T) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "44444444-4444-4444-4444-444444444444",
		"iat": time.Now().UTC().Unix(),
		"exp": time.Now().UTC().Add(time.Hour).Unix(),
	})
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(i.priv)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + signed
}

func TestListRequiresToken(t *testing.T) {
	mux, _ := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/spaces")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body errorBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "UNAUTHORIZED" || body.TraceID == "" {
		t.Fatalf("body %+v", body)
	}
}

func TestListEnvelope(t *testing.T) {
	mux, keys := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var env listEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data == nil || env.Meta.Page != 1 || env.Meta.Limit != 20 || env.Meta.Total != 4 {
		t.Fatalf("envelope %+v", env)
	}
	if env.Data[0].Name != "Aula Magna" || env.Data[0].Kind != "AUDITORIUM" {
		t.Fatalf("order/camelCase %+v", env.Data)
	}
	norte := env.Data[3]
	if norte.ID != "11111111-1111-1111-1111-111111111111" || norte.Available != true {
		t.Fatalf("sala norte %+v", norte)
	}
}

func TestGetMalformedUUID(t *testing.T) {
	mux, keys := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/not-a-uuid", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "VALIDATION_ERROR" {
		t.Fatalf("body %+v", body)
	}
}

func TestGetMissingSpace(t *testing.T) {
	mux, keys := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/99999999-9999-9999-9999-999999999999", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body errorBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "NOT_FOUND" {
		t.Fatalf("body %+v", body)
	}
}

func TestRejectsHS256AndNone(t *testing.T) {
	mux, _ := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "44444444-4444-4444-4444-444444444444",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	hs.Header["kid"] = testKID
	signed, err := hs.SignedString([]byte("not-rsa"))
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("hs256 %d", res.StatusCode)
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"44444444-4444-4444-4444-444444444444","exp":4102444800}`))
	none := header + "." + payload + "."
	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces", nil)
	req2.Header.Set("Authorization", "Bearer "+none)
	res2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("none %d", res2.StatusCode)
	}
}

func TestHealth(t *testing.T) {
	mux, _ := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body healthBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Service != "space-service" || body.Timestamp == "" {
		t.Fatalf("body %+v", body)
	}
}

func TestGetInactiveSpaceStillReturned(t *testing.T) {
	mux, keys := spaceMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/55555555-5555-5555-5555-555555555555", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var dto spaceDTO
	if err := json.NewDecoder(res.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto.Available != false || dto.Name != "Oficina Cerrada" {
		t.Fatalf("dto %+v", dto)
	}
}
