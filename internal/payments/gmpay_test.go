package payments

import "testing"

func TestGMPaySignatureOfficialExample(t *testing.T) {
	params := map[string]string{
		"pid": "1000",
		"order_id": "ORD202605230001",
		"currency": "cny",
		"token": "usdt",
		"network": "tron",
		"amount": "100",
		"notify_url": "https://merchant.example/notify",
		"redirect_url": "https://merchant.example/return",
		"name": "VIP",
	}
	const want = "6f874b1919d95081835e2809b620e354a5866f5a6dbb2e432d1627f1eb10059d"
	if got := GMPaySignature(params, "epusdt_secret_key"); got != want {
		t.Fatalf("GMPaySignature() = %s, want %s", got, want)
	}
	params["signature"] = want
	if !VerifyGMPaySignature(params, "epusdt_secret_key") {
		t.Fatal("official example signature did not verify")
	}
}

func TestGMPaySignatureOmitsEmptyAndSignatureFields(t *testing.T) {
	first := map[string]string{"pid":"1000","empty":"","signature":"ignored","amount":"1"}
	second := map[string]string{"amount":"1","pid":"1000"}
	if GMPaySignature(first,"secret") != GMPaySignature(second,"secret") {
		t.Fatal("empty and signature fields must be excluded")
	}
}

func TestVerifyGMPaySignatureRejectsTampering(t *testing.T) {
	params := map[string]string{"pid":"1000","amount":"10"}
	params["signature"] = GMPaySignature(params,"secret")
	if !VerifyGMPaySignature(params,"secret") { t.Fatal("valid signature rejected") }
	params["amount"] = "11"
	if VerifyGMPaySignature(params,"secret") { t.Fatal("tampered payload accepted") }
	if VerifyGMPaySignature(map[string]string{"signature":"bad"},"secret") { t.Fatal("malformed signature accepted") }
}
