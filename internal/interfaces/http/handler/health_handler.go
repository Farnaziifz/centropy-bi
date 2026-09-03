package handler

import "net/http"

// Health is an unauthenticated liveness probe.
//
//	@Summary		Liveness probe
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/healthz [get]
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
