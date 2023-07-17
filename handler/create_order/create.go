package order_handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ansakharov/lets_test/internal/pkg/entity/order"
)

var (
	ErrInvalidUserID      = errors.New("invalid user OrderID")
	ErrInvalidAmount      = errors.New("invalid price")
	ErrInvalidPaymentType = errors.New("invalid payment type")
	ErrEmptyItems         = errors.New("items can't be empty")
	ErrInvalidItemID      = errors.New("invalid service id")
)

// OrderFromDTO creates Order for business layer.
func (in OrderIn) OrderFromDTO() order.Order {
	items := make([]order.Item, 0, len(in.Items))
	for _, item := range in.Items {
		items = append(items, order.Item{
			ID:               item.ID,
			Amount:           item.Amount,
			DiscountedAmount: item.Discount,
		})
	}

	return order.Order{
		Status:      order.CreatedStatus,
		UserID:      in.UserID,
		PaymentType: order.PaymentType(paymentTypes[in.PaymentType]),
		Items:       items,
	}
}

// Create responsible for saving new order.
func (h Handler) Create(ctx context.Context) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// prepare dto to parse request
		in := &OrderIn{}
		// parse req body to dto
		err := json.NewDecoder(r.Body).Decode(&in)
		if err != nil {
			h.log.Errorf("can't parse req: %s", err.Error())
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}

		// check that request valid
		err = h.validateReq(in)
		if err != nil {
			h.log.Errorf("bad req: %v: %s", in, err.Error())
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		saveOrder := in.OrderFromDTO()
		err = h.uCase.Save(ctx, h.log, &saveOrder)
		if err != nil {
			h.log.Errorf("uCase.Save: %v", err)
			http.Error(w, "can't create saveOrder: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		m := make(map[string]interface{})
		m["success"] = "ok"

		err = json.NewEncoder(w).Encode(m)
		if err != nil {
			h.log.Errorf("Encode: %v", err)
		}
	}
	return http.HandlerFunc(fn)
}
