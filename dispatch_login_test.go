package main

import (
	"context"
	"errors"
	"testing"
)

type recordedSMS struct {
	verifyErr error
	verified  bool
}

func (r *recordedSMS) SendCode(context.Context, string, string) error { return nil }
func (r *recordedSMS) VerifyCode(context.Context, string, string, string) error {
	r.verified = true
	return r.verifyErr
}

func TestTechnicianConfirmationControlsDispatchState(t *testing.T) {
	tests := []struct {
		name          string
		technician    string
		verifyErr     error
		wantStatus    string
		wantVerified  bool
		wantOrderView bool
	}{
		{name: "assigned technician confirms", technician: "tech-17", wantStatus: statusTechnicianConfirmed, wantVerified: true, wantOrderView: true},
		{name: "rejected code retains dispatch", technician: "tech-17", verifyErr: errors.New("code rejected"), wantStatus: statusDispatched, wantVerified: true},
		{name: "other technician never reaches verification", technician: "tech-99", wantStatus: statusDispatched},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sms := &recordedSMS{verifyErr: tt.verifyErr}
			order := &WorkOrder{ID: "WO-1042", Technician: "tech-17", Phone: "+15551234567", Status: statusDispatched}
			login := &dispatchLogin{sms: sms, orders: map[string]*WorkOrder{order.ID: order}}

			view, err := login.confirm(context.Background(), order.ID, tt.technician, "381204", "attempt-42")
			if (view != nil) != tt.wantOrderView || (err == nil) != tt.wantOrderView {
				t.Fatalf("confirm() view = %v, error = %v", view, err)
			}
			if order.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", order.Status, tt.wantStatus)
			}
			if sms.verified != tt.wantVerified {
				t.Fatalf("VerifyCode called = %v, want %v", sms.verified, tt.wantVerified)
			}
		})
	}
}
