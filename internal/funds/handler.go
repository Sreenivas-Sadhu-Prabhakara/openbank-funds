package funds

import (
	"net/http"

	"github.com/sreeni/openbank-bian/pkg/httpx"
	"github.com/sreeni/openbank-bian/pkg/obie"
)

// Handler exposes the funds (CBPII) service over HTTP using OBIE
// request/response shapes. baseURL is used to build absolute Self links.
type Handler struct {
	svc     *Service
	baseURL string
}

// NewHandler constructs the HTTP handler.
func NewHandler(svc *Service, baseURL string) *Handler {
	return &Handler{svc: svc, baseURL: baseURL}
}

// Routes registers every funds route on a ServeMux and returns it.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Confirmation of funds (CBPII). Only a POST is exposed — there is no public
	// GET in the OBIE CBPII API.
	mux.HandleFunc("POST /funds-confirmations", h.createFundsConfirmation)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func (h *Handler) createFundsConfirmation(w http.ResponseWriter, r *http.Request) {
	var req fundsConfirmationReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.RespondError(w, err)
		return
	}

	fc, err := h.svc.Create(r.Context(), CreateInput{
		ConsentID:        req.Data.ConsentID,
		Reference:        req.Data.Reference,
		InstructedAmount: req.Data.InstructedAmount,
	})
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	self := h.baseURL + "/funds-confirmations/" + fc.FundsConfirmationID
	httpx.WriteJSON(w, http.StatusCreated, obie.NewResponse(self, fundsConfirmationData(fc)))
}
