package funds

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// newTestHandler wires a handler over an in-memory repo and the given fake
// upstreams, with the deterministic clock/ids from newTestService.
func newTestHandler(fc *fakeConsent, ck *fakeChecker) http.Handler {
	return NewHandler(newTestService(fc, ck), "http://funds.test").Routes()
}

// do issues a request to the handler and returns the recorder.
func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const confirmationBody = `{
	"Data": {
		"ConsentId": "cons-1",
		"Reference": "Purchase01",
		"InstructedAmount": {"Amount": "20.00", "Currency": "GBP"}
	}
}`

func TestFundsConfirmationAvailableOverHTTP(t *testing.T) {
	fc := &fakeConsent{view: authorisedView("70000170000001")}
	ck := &fakeChecker{available: true}
	h := newTestHandler(fc, ck)

	w := do(t, h, http.MethodPost, "/funds-confirmations", confirmationBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", w.Code, w.Body)
	}

	var created struct {
		Data  fundsConfirmationRespData `json:"Data"`
		Links obie.Links                `json:"Links"`
	}
	mustDecode(t, w, &created)

	if !created.Data.FundsAvailableResult.FundsAvailable {
		t.Fatalf("FundsAvailable = false, want true")
	}
	if created.Data.FundsAvailableResult.FundsAvailableDateTime == "" {
		t.Fatal("missing FundsAvailableDateTime")
	}
	if created.Data.InstructedAmount.String() != "20" {
		t.Fatalf("amount = %s", created.Data.InstructedAmount)
	}
	if created.Data.Reference != "Purchase01" {
		t.Fatalf("reference = %s", created.Data.Reference)
	}
	id := created.Data.FundsConfirmationID
	if id == "" {
		t.Fatal("missing FundsConfirmationId")
	}
	if created.Links.Self != "http://funds.test/funds-confirmations/"+id {
		t.Fatalf("Self = %s", created.Links.Self)
	}
	if ck.gotIdent != "70000170000001" {
		t.Fatalf("checked identification = %q, want the consent's debtor account", ck.gotIdent)
	}
}

func TestFundsConfirmationUnavailableOverHTTP(t *testing.T) {
	fc := &fakeConsent{view: authorisedView("70000170000001")}
	ck := &fakeChecker{available: false}
	h := newTestHandler(fc, ck)

	w := do(t, h, http.MethodPost, "/funds-confirmations", confirmationBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", w.Code, w.Body)
	}
	var created struct {
		Data fundsConfirmationRespData `json:"Data"`
	}
	mustDecode(t, w, &created)
	if created.Data.FundsAvailableResult.FundsAvailable {
		t.Fatalf("FundsAvailable = true, want false")
	}
	if created.Data.FundsAvailableResult.FundsAvailableDateTime == "" {
		t.Fatal("missing FundsAvailableDateTime even on a negative result")
	}
}

func TestCreateWrongConsentTypeOverHTTP(t *testing.T) {
	view := authorisedView("acc-1")
	view.Type = "account-access"
	h := newTestHandler(&fakeConsent{view: view}, &fakeChecker{available: true})

	w := do(t, h, http.MethodPost, "/funds-confirmations", confirmationBody)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body)
	}
	var errBody obie.ErrorResponse
	mustDecode(t, w, &errBody)
	if errBody.Code != "Forbidden" || len(errBody.Errors) == 0 {
		t.Fatalf("unexpected error body %+v", errBody)
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	h := newTestHandler(&fakeConsent{view: authorisedView("acc-1")}, &fakeChecker{available: true})
	w := do(t, h, http.MethodPost, "/funds-confirmations",
		`{"Data":{"ConsentId":"cons-1","InstructedAmount":{"Amount":"1.00","Currency":"GBP"}},"Bogus":true}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body)
	}
}

func TestHealth(t *testing.T) {
	h := newTestHandler(&fakeConsent{}, &fakeChecker{})
	w := do(t, h, http.MethodGet, "/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("health status = %d", w.Code)
	}
}

func mustDecode(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
}
