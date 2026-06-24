package funds

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sreeni/openbank-bian/pkg/accountscli"
	"github.com/sreeni/openbank-bian/pkg/consentcli"
	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// ConsentClient is the slice of the consent service this domain depends on:
// fetch a consent view to validate the funds-confirmation consent the PSU
// authorised. *consentcli.Client satisfies it; tests inject a fake so the
// consent service need not be running.
type ConsentClient interface {
	Get(ctx context.Context, id string) (*consentcli.View, error)
}

// FundsChecker is the slice of the accounts service this domain depends on: ask
// whether a debtor account holds at least a given amount. *accountscli.Client
// satisfies it; tests inject a fake so the accounts service need not be running.
type FundsChecker interface {
	FundsAvailable(ctx context.Context, identification string, amount obie.Amount) (bool, error)
}

// Service holds the confirmation-of-funds business logic. The consent client,
// funds checker, clock and id generator are injected so tests are deterministic
// and the upstream services can be faked.
type Service struct {
	repo     Repository
	consent  ConsentClient
	accounts FundsChecker
	now      func() time.Time
	newID    func() string
}

// NewService wires a Service to its repository, consent client and funds checker
// using a real clock and UUID ids.
func NewService(repo Repository, consent ConsentClient, accounts FundsChecker) *Service {
	return &Service{
		repo:     repo,
		consent:  consent,
		accounts: accounts,
		now:      time.Now,
		newID:    uuid.NewString,
	}
}

// CreateInput carries everything needed to perform a confirmation of funds, kept
// independent of the OBIE wire shapes.
type CreateInput struct {
	ConsentID        string
	Reference        string
	InstructedAmount obie.Amount
}

// Create performs a confirmation of funds against an authorised
// funds-confirmation consent.
//
// The flow is, in order:
//  1. Validate the request: ConsentId and a well-formed InstructedAmount are
//     required.
//  2. Validate the consent: it must exist, be of type funds-confirmation and be
//     Authorised. An unknown consent is a 400; a wrong-type or unauthorised
//     consent is a 403.
//  3. Ask the accounts service whether the consent's debtor account holds the
//     instructed amount. An unknown account is a 422; any other checker failure
//     is surfaced as a 502 (the upstream is unreachable/erroring).
//  4. Persist the confirmation (for audit) and return it. A negative result is
//     still a successful confirmation.
func (s *Service) Create(ctx context.Context, in CreateInput) (*FundsConfirmation, error) {
	if in.ConsentID == "" {
		return nil, httpx.BadRequest("ConsentId is required",
			httpx.Detail(obie.ErrFieldMissing, "missing ConsentId", "Data.ConsentId"))
	}
	if err := in.InstructedAmount.Validate(); err != nil {
		return nil, httpx.BadRequest("Invalid InstructedAmount",
			httpx.Detail(obie.ErrFieldInvalid, err.Error(), "Data.InstructedAmount"))
	}

	view, err := s.consent.Get(ctx, in.ConsentID)
	if err != nil {
		if errors.Is(err, consentcli.ErrNotFound) {
			return nil, httpx.BadRequest("Consent not found",
				httpx.Detail(obie.ErrResourceNotFound, "no such consent", "Data.ConsentId"))
		}
		return nil, httpx.Internal("could not load consent")
	}
	if view.Type != consentcli.TypeFundsConfirmation {
		return nil, httpx.Forbidden("Consent is not a funds-confirmation consent",
			httpx.Detail(obie.ErrResourceInvalid, "unexpected consent type: "+view.Type, "Data.ConsentId"))
	}
	if view.Status != consentcli.StatusAuthorised {
		return nil, httpx.Forbidden("Consent is not authorised",
			httpx.Detail(obie.ErrResourceInvalid, "consent status is "+view.Status, "Data.ConsentId"))
	}

	// The account to check is the one the PSU named on the consent, never one the
	// caller supplies — funds may only be confirmed against the consented debtor.
	available, err := s.accounts.FundsAvailable(ctx, view.DebtorAccountID, in.InstructedAmount)
	if err != nil {
		if errors.Is(err, accountscli.ErrAccountNotFound) {
			return nil, httpx.Unprocessable("Debtor account not found",
				httpx.Detail(obie.ErrResourceInvalid, "the consent's debtor account is unknown", "Data.ConsentId"))
		}
		return nil, &httpx.APIError{Status: 502, Message: "could not confirm funds with the accounts service"}
	}

	now := s.now()
	fc := &FundsConfirmation{
		FundsConfirmationID:    s.newID(),
		ConsentID:              in.ConsentID,
		CreationDateTime:       now,
		Reference:              in.Reference,
		InstructedAmount:       in.InstructedAmount,
		FundsAvailable:         available,
		FundsAvailableDateTime: now,
	}
	if err := s.repo.Create(ctx, fc); err != nil {
		return nil, httpx.Internal("could not persist funds confirmation")
	}
	return fc, nil
}

// Get returns a recorded funds confirmation by id, mapping a missing record to a
// 404. CBPII has no public GET endpoint; this backs the audit/round-trip path.
func (s *Service) Get(ctx context.Context, id string) (*FundsConfirmation, error) {
	fc, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("Funds confirmation not found",
				httpx.Detail(obie.ErrResourceNotFound, "no such funds confirmation", ""))
		}
		return nil, httpx.Internal("could not load funds confirmation")
	}
	return fc, nil
}
