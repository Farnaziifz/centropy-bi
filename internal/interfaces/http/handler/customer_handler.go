package handler

import (
	"net/http"

	customercmd "centropy-affilate/internal/application/customer/command"
	customerquery "centropy-affilate/internal/application/customer/query"
	"centropy-affilate/internal/domain/customer"
	"centropy-affilate/internal/interfaces/http/dto"
	"centropy-affilate/pkg/cqrs"
)

type CustomerHandler struct {
	bus *cqrs.Bus
}

func NewCustomerHandler(bus *cqrs.Bus) *CustomerHandler {
	return &CustomerHandler{bus: bus}
}

// Sync pulls the full AlefGym user directory into the local Customer
// table. See internal/application/customer/command.SyncCustomersHandler.
//
//	@Summary		Sync customers from AlefGym
//	@Description	Pulls the full AlefGym user directory into the local Customer table.
//	@Tags			customers
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	customercmd.SyncCustomersResult
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/admin/customers/sync [post]
func (h *CustomerHandler) Sync(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteCommand[customercmd.SyncCustomersCommand, customercmd.SyncCustomersResult](
		r.Context(), h.bus, customercmd.SyncCustomersCommand{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// List returns the locally synced customer directory.
//
//	@Summary		List customers
//	@Description	Returns the locally synced customer directory.
//	@Tags			customers
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		customer.Customer
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/admin/customers [get]
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteQuery[customerquery.ListCustomersQuery, []customer.Customer](
		r.Context(), h.bus, customerquery.ListCustomersQuery{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}
