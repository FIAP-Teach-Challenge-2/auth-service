package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func (a *App) chaosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.Fault.Enable()
		log.Println("Chaos: falha injetada no /validate (respostas 500 até o restart do pod)")
	case http.MethodDelete:
		a.Fault.Disable()
		log.Println("Chaos: falha removida do /validate")
	case http.MethodGet:
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"fault_enabled": a.Fault.Enabled()})
}
