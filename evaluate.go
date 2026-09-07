package main

import "net/http"

func handleEvaluateFlag(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
