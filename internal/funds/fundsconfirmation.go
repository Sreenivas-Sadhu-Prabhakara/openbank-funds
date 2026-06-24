// Package funds implements the BIAN "Confirmation of Funds" service domain: the
// OBIE CBPII flow that answers, for a single funds-confirmation consent, whether
// a debtor account holds at least a requested amount. It owns no balance data of
// its own — it validates the caller's consent against the consent service (via
// consentcli) and asks the accounts service (via accountscli) for the actual
// availability. The created confirmations are persisted purely for audit, since
// CBPII exposes only a POST and no public GET.
package funds

import (
	"time"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// FundsConfirmation is the aggregate root: a single recorded confirmation that a
// debtor account either did or did not hold the instructed amount at the moment
// the check ran. CBPII confirmations are immutable once created.
type FundsConfirmation struct {
	FundsConfirmationID string
	ConsentID           string
	CreationDateTime    time.Time

	// Reference is the optional free-text reference echoed back from the request.
	Reference string

	// InstructedAmount is the sum whose availability was checked.
	InstructedAmount obie.Amount

	// FundsAvailable is the outcome of the check and FundsAvailableDateTime is the
	// instant it was determined. A false result is still a successful confirmation
	// — the question "are funds available?" was answered, the answer was just "no".
	FundsAvailable         bool
	FundsAvailableDateTime time.Time
}
