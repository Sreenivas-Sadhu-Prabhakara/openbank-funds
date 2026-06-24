package funds

import (
	"time"

	"github.com/sreeni/openbank-bian/pkg/obie"
)

// ---- funds-confirmation (CBPII) wire shapes ----

// fundsConfirmationReq is the OBIE OBFundsConfirmation1 request body. Unlike
// consent and payment resources, a CBPII funds-confirmation carries no Risk
// block, so none is accepted here.
type fundsConfirmationReq struct {
	Data struct {
		ConsentID        string      `json:"ConsentId"`
		Reference        string      `json:"Reference"`
		InstructedAmount obie.Amount `json:"InstructedAmount"`
	} `json:"Data"`
}

// fundsAvailableResult is the OBIE OBFundsAvailableResult1 sub-object reporting
// the outcome of the check.
type fundsAvailableResult struct {
	FundsAvailableDateTime string `json:"FundsAvailableDateTime"`
	FundsAvailable         bool   `json:"FundsAvailable"`
}

// fundsConfirmationRespData is the OBIE OBFundsConfirmationResponse1 Data block.
type fundsConfirmationRespData struct {
	FundsConfirmationID  string               `json:"FundsConfirmationId"`
	ConsentID            string               `json:"ConsentId"`
	CreationDateTime     string               `json:"CreationDateTime"`
	Reference            string               `json:"Reference,omitempty"`
	InstructedAmount     obie.Amount          `json:"InstructedAmount"`
	FundsAvailableResult fundsAvailableResult `json:"FundsAvailableResult"`
}

func fundsConfirmationData(fc *FundsConfirmation) fundsConfirmationRespData {
	return fundsConfirmationRespData{
		FundsConfirmationID: fc.FundsConfirmationID,
		ConsentID:           fc.ConsentID,
		CreationDateTime:    rfc3339(fc.CreationDateTime),
		Reference:           fc.Reference,
		InstructedAmount:    fc.InstructedAmount,
		FundsAvailableResult: fundsAvailableResult{
			FundsAvailableDateTime: rfc3339(fc.FundsAvailableDateTime),
			FundsAvailable:         fc.FundsAvailable,
		},
	}
}

// ---- shared helpers ----

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }
