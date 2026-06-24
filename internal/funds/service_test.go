package funds

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sreeni/openbank-bian/pkg/accountscli"
	"github.com/sreeni/openbank-bian/pkg/consentcli"
	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// fakeConsent is a test double for the consent service. It returns a fixed view
// (or error) so tests need not run the real consent service.
type fakeConsent struct {
	view   *consentcli.View
	getErr error
}

func (f *fakeConsent) Get(_ context.Context, _ string) (*consentcli.View, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.view, nil
}

// fakeChecker is a test double for the accounts service's funds check. It
// records the identification it was asked about so tests can assert the consent's
// debtor account is the one checked.
type fakeChecker struct {
	available bool
	err       error
	calls     int
	gotIdent  string
	gotAmount obie.Amount
}

func (f *fakeChecker) FundsAvailable(_ context.Context, identification string, amount obie.Amount) (bool, error) {
	f.calls++
	f.gotIdent = identification
	f.gotAmount = amount
	return f.available, f.err
}

// authorisedView builds a funds-confirmation consent view in Authorised status
// naming the given debtor account, ready to back a successful confirmation.
func authorisedView(debtorAccountID string) *consentcli.View {
	return &consentcli.View{
		ConsentID:       "cons-1",
		Type:            consentcli.TypeFundsConfirmation,
		Status:          consentcli.StatusAuthorised,
		DebtorAccountID: debtorAccountID,
	}
}

// newTestService returns a service backed by an in-memory repo and the given
// fake upstreams, with a fixed clock and deterministic ids.
func newTestService(fc *fakeConsent, ck *fakeChecker) *Service {
	s := NewService(NewMemRepository(), fc, ck)
	fixed := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return fixed }
	n := 0
	s.newID = func() string { n++; return "fc-" + itoa(n) }
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *httpx.APIError, got %v", err)
	}
	if apiErr.Status != status {
		t.Fatalf("status = %d, want %d (%s)", apiErr.Status, status, apiErr.Message)
	}
}

// validInput is a baseline CreateInput that pairs with authorisedView.
func validInput() CreateInput {
	return CreateInput{
		ConsentID:        "cons-1",
		Reference:        "REF-9",
		InstructedAmount: obie.MustAmount("20.00", "GBP"),
	}
}

func TestCreateHappyPathFundsAvailable(t *testing.T) {
	ctx := context.Background()
	fc := &fakeConsent{view: authorisedView("70000170000001")}
	ck := &fakeChecker{available: true}
	s := newTestService(fc, ck)

	got, err := s.Create(ctx, validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.FundsConfirmationID != "fc-1" || got.ConsentID != "cons-1" {
		t.Fatalf("unexpected confirmation %+v", got)
	}
	if !got.FundsAvailable {
		t.Fatalf("FundsAvailable = false, want true")
	}
	if got.FundsAvailableDateTime.IsZero() {
		t.Fatal("FundsAvailableDateTime not set")
	}

	// The checker must be asked about the consent's debtor account and the
	// requested amount — never an account the caller supplies.
	if ck.calls != 1 {
		t.Fatalf("FundsAvailable called %d times, want 1", ck.calls)
	}
	if ck.gotIdent != "70000170000001" {
		t.Fatalf("checked identification = %q, want the consent's debtor account", ck.gotIdent)
	}
	if ck.gotAmount.String() != "20" || ck.gotAmount.Currency != "GBP" {
		t.Fatalf("checked amount = %s %s", ck.gotAmount, ck.gotAmount.Currency)
	}

	// The confirmation is persisted for audit and readable back.
	reloaded, err := s.Get(ctx, got.FundsConfirmationID)
	if err != nil || reloaded.FundsConfirmationID != got.FundsConfirmationID {
		t.Fatalf("get: err=%v got=%+v", err, reloaded)
	}
}

func TestCreateHappyPathFundsUnavailable(t *testing.T) {
	ctx := context.Background()
	fc := &fakeConsent{view: authorisedView("70000170000001")}
	ck := &fakeChecker{available: false}
	s := newTestService(fc, ck)

	got, err := s.Create(ctx, validInput())
	if err != nil {
		t.Fatalf("create: %v", err) // a negative result is still a successful confirmation
	}
	if got.FundsAvailable {
		t.Fatalf("FundsAvailable = true, want false")
	}
	if ck.calls != 1 || ck.gotIdent != "70000170000001" {
		t.Fatalf("checker not called with the consent's debtor account: calls=%d ident=%q", ck.calls, ck.gotIdent)
	}
}

func TestCreateRequiresConsentID(t *testing.T) {
	ctx := context.Background()
	ck := &fakeChecker{available: true}
	s := newTestService(&fakeConsent{view: authorisedView("acc-1")}, ck)

	in := validInput()
	in.ConsentID = ""
	_, err := s.Create(ctx, in)
	wantStatus(t, err, http.StatusBadRequest)
	if ck.calls != 0 {
		t.Fatal("FundsAvailable should not be called without a consent id")
	}
}

func TestCreateUnknownConsentIsBadRequest(t *testing.T) {
	ctx := context.Background()
	ck := &fakeChecker{available: true}
	s := newTestService(&fakeConsent{getErr: consentcli.ErrNotFound}, ck)

	_, err := s.Create(ctx, validInput())
	wantStatus(t, err, http.StatusBadRequest)
	if ck.calls != 0 {
		t.Fatal("FundsAvailable should not be called for an unknown consent")
	}
}

func TestCreateRejectsWrongConsentType(t *testing.T) {
	ctx := context.Background()
	view := authorisedView("acc-1")
	view.Type = consentcli.TypeAccountAccess
	ck := &fakeChecker{available: true}
	s := newTestService(&fakeConsent{view: view}, ck)

	_, err := s.Create(ctx, validInput())
	wantStatus(t, err, http.StatusForbidden)
	if ck.calls != 0 {
		t.Fatal("FundsAvailable should not be called for the wrong consent type")
	}
}

func TestCreateRejectsUnauthorisedConsent(t *testing.T) {
	ctx := context.Background()
	view := authorisedView("acc-1")
	view.Status = consentcli.StatusAwaitingAuthorisation
	ck := &fakeChecker{available: true}
	s := newTestService(&fakeConsent{view: view}, ck)

	_, err := s.Create(ctx, validInput())
	wantStatus(t, err, http.StatusForbidden)
	if ck.calls != 0 {
		t.Fatal("FundsAvailable should not be called for an unauthorised consent")
	}
}

func TestCreateUnknownAccountIsUnprocessable(t *testing.T) {
	ctx := context.Background()
	fc := &fakeConsent{view: authorisedView("acc-gone")}
	ck := &fakeChecker{err: accountscli.ErrAccountNotFound}
	s := newTestService(fc, ck)

	_, err := s.Create(ctx, validInput())
	wantStatus(t, err, http.StatusUnprocessableEntity)
	if ck.gotIdent != "acc-gone" {
		t.Fatalf("checked identification = %q, want the consent's debtor account", ck.gotIdent)
	}
}

func TestCreateCheckerErrorIsBadGateway(t *testing.T) {
	ctx := context.Background()
	fc := &fakeConsent{view: authorisedView("acc-1")}
	ck := &fakeChecker{err: errors.New("accounts service returned 500")}
	s := newTestService(fc, ck)

	_, err := s.Create(ctx, validInput())
	wantStatus(t, err, http.StatusBadGateway)
}

func TestGetUnknownIsNotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestService(&fakeConsent{}, &fakeChecker{})
	_, err := s.Get(ctx, "nope")
	wantStatus(t, err, http.StatusNotFound)
}
