package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
)

type loginInput struct {
	WorkOrderID string `json:"work_order_id"`
	Technician  string `json:"technician_id"`
	AttemptID   string `json:"attempt_id"`
	Code        string `json:"code"`
}

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	login := &dispatchLogin{
		sms: newSMSClient(apiKey),
		orders: map[string]*WorkOrder{
			"WO-1042": {
				ID: "WO-1042", Technician: "tech-17", Phone: "+15551234567", Status: statusDispatched,
				Photos:   []WorkPhoto{{ObjectKey: "work-orders/WO-1042/panel.jpg", Caption: "Control panel before service"}},
				FollowUp: FollowUp{Required: true, Note: "Confirm replacement relay inventory"},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", func(w http.ResponseWriter, r *http.Request) {
		input, ok := decodeInput(w, r, false)
		if !ok {
			return
		}
		if err := login.requestCode(r.Context(), input.WorkOrderID, input.Technician, input.AttemptID); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "code_sent"})
	})
	mux.HandleFunc("POST /login/verify", func(w http.ResponseWriter, r *http.Request) {
		input, ok := decodeInput(w, r, true)
		if !ok {
			return
		}
		order, err := login.confirm(r.Context(), input.WorkOrderID, input.Technician, input.Code, input.AttemptID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	log.Println("field dispatch login listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func decodeInput(w http.ResponseWriter, r *http.Request, codeRequired bool) (loginInput, bool) {
	var input loginInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return loginInput{}, false
	}
	if input.WorkOrderID == "" || input.Technician == "" || input.AttemptID == "" || (codeRequired && input.Code == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "required field is empty"})
		return loginInput{}, false
	}
	return input, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	var apiErr *InfraiError
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		status = apiErr.Status
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
