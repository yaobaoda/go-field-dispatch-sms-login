package main

import (
	"context"
	"errors"
)

const (
	statusDispatched          = "dispatched"
	statusTechnicianConfirmed = "technician_confirmed"
)

type WorkPhoto struct {
	ObjectKey string `json:"object_key"`
	Caption   string `json:"caption"`
}

type FollowUp struct {
	Required bool   `json:"required"`
	Note     string `json:"note"`
}

type WorkOrder struct {
	ID         string      `json:"work_order_id"`
	Technician string      `json:"technician_id"`
	Phone      string      `json:"-"`
	Status     string      `json:"status"`
	Photos     []WorkPhoto `json:"photos"`
	FollowUp   FollowUp    `json:"follow_up"`
}

type codeSender interface {
	SendCode(context.Context, string, string) error
	VerifyCode(context.Context, string, string, string) error
}

type dispatchLogin struct {
	sms    codeSender
	orders map[string]*WorkOrder
}

func (d *dispatchLogin) requestCode(ctx context.Context, orderID, technicianID, attemptID string) error {
	order, err := d.assignedOrder(orderID, technicianID)
	if err != nil {
		return err
	}
	return d.sms.SendCode(ctx, order.Phone, "send:"+attemptID)
}

func (d *dispatchLogin) confirm(ctx context.Context, orderID, technicianID, code, attemptID string) (*WorkOrder, error) {
	order, err := d.assignedOrder(orderID, technicianID)
	if err != nil {
		return nil, err
	}
	if err := d.sms.VerifyCode(ctx, order.Phone, code, "verify:"+attemptID); err != nil {
		return nil, err
	}
	order.Status = statusTechnicianConfirmed
	return order, nil
}

func (d *dispatchLogin) assignedOrder(orderID, technicianID string) (*WorkOrder, error) {
	order, ok := d.orders[orderID]
	if !ok || order.Technician != technicianID {
		return nil, errors.New("work order assignment does not match")
	}
	return order, nil
}
