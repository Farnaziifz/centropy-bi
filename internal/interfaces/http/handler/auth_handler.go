package handler

import (
	"net/http"

	authcmd "centropy-affilate/internal/application/auth/command"
	"centropy-affilate/internal/interfaces/http/dto"
	"centropy-affilate/pkg/cqrs"
)

type AuthHandler struct {
	bus *cqrs.Bus
}

func NewAuthHandler(bus *cqrs.Bus) *AuthHandler {
	return &AuthHandler{bus: bus}
}

// Login authenticates an admin/ops user and returns a bearer token.
//
//	@Summary		Admin login
//	@Description	Authenticates an admin/ops user and returns a bearer token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.LoginRequest	true	"credentials"
//	@Success		200		{object}	authcmd.LoginResult
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	req, err := dto.DecodeAndValidate[dto.LoginRequest](r)
	if err != nil {
		dto.WriteError(w, err)
		return
	}

	result, err := cqrs.ExecuteCommand[authcmd.LoginCommand, authcmd.LoginResult](
		r.Context(), h.bus, authcmd.LoginCommand{Email: req.Email, Password: req.Password},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}

	dto.WriteJSON(w, http.StatusOK, result)
}
