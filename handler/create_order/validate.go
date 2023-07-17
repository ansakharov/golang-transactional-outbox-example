package order_handler

// validates request.
func (h Handler) validateReq(in *OrderIn) error {
	// user OrderID can't be 0
	if in.UserID == 0 {
		return ErrInvalidUserID
	}
	// payment type must be in paymentTypes
	if _, ok := paymentTypes[in.PaymentType]; !ok {
		return ErrInvalidPaymentType
	}
	// no services passed in request
	if len(in.Items) == 0 {
		return ErrEmptyItems
	}
	// service doesn't contain valid id
	for i := range in.Items {
		if in.Items[i].ID == 0 {
			return ErrInvalidItemID
		}

		if in.Items[i].Amount == 0 {
			return ErrInvalidAmount
		}
	}
	return nil
}
