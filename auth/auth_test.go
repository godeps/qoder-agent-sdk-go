package auth

import "testing"

func TestBrandEnvVars(t *testing.T) {
	cases := []struct {
		brand   Brand
		wantTok string
		wantSA  string
	}{
		{BrandGlobal, "QODER_PERSONAL_ACCESS_TOKEN", "QODER_SERVICE_ACCOUNT_KEY"},
		{BrandCN, "QODERCN_PERSONAL_ACCESS_TOKEN", "QODERCN_SERVICE_ACCOUNT_KEY"},
		{BrandAuto, "", ""},
	}
	for _, c := range cases {
		if got := AccessTokenEnvVar(c.brand); got != c.wantTok {
			t.Errorf("AccessTokenEnvVar(%q) = %q, want %q", c.brand, got, c.wantTok)
		}
		if got := ServiceAccountEnvVar(c.brand); got != c.wantSA {
			t.Errorf("ServiceAccountEnvVar(%q) = %q, want %q", c.brand, got, c.wantSA)
		}
	}
}

func TestEnvAccessTokenAutoDetectsEitherBrand(t *testing.T) {
	t.Setenv("QODERCN_PERSONAL_ACCESS_TOKEN", "")
	t.Setenv("QODER_PERSONAL_ACCESS_TOKEN", "")

	// Neither set → empty.
	if tok := EnvAccessToken(BrandAuto); tok != "" {
		t.Fatalf("EnvAccessToken(auto) with none set = %q, want empty", tok)
	}

	// Global set → global token.
	t.Setenv("QODER_PERSONAL_ACCESS_TOKEN", "global-tok")
	if tok := EnvAccessToken(BrandAuto); tok != "global-tok" {
		t.Fatalf("EnvAccessToken(auto) global = %q, want global-tok", tok)
	}

	// cn set takes precedence over global in auto mode.
	t.Setenv("QODERCN_PERSONAL_ACCESS_TOKEN", "cn-tok")
	if tok := EnvAccessToken(BrandAuto); tok != "cn-tok" {
		t.Fatalf("EnvAccessToken(auto) cn-precedence = %q, want cn-tok", tok)
	}

	// Explicit brand only reads its own var.
	if tok := EnvAccessToken(BrandGlobal); tok != "global-tok" {
		t.Fatalf("EnvAccessToken(global) = %q, want global-tok", tok)
	}
	if tok := EnvAccessToken(BrandCN); tok != "cn-tok" {
		t.Fatalf("EnvAccessToken(cn) = %q, want cn-tok", tok)
	}
}

func TestPayloadAccessTokenBrand(t *testing.T) {
	t.Setenv("QODER_PERSONAL_ACCESS_TOKEN", "gtok")
	t.Setenv("QODERCN_PERSONAL_ACCESS_TOKEN", "")
	p, err := AccessTokenFromEnvBrand(BrandGlobal).Payload()
	if err != nil {
		t.Fatalf("Payload: %v", err)
	}
	if p.AccessToken != "gtok" {
		t.Fatalf("access token = %q, want gtok", p.AccessToken)
	}
}
