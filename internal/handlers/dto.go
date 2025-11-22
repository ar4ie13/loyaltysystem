package handlers

type userOrdersResponse struct {
	OrderNumber string   `json:"number" db:"order_num"`
	Status      string   `json:"status" db:"status"`
	Accrual     *float64 `json:"accrual,omitempty" db:"accrual"`
	CreatedAt   string   `json:"uploaded_at" db:"created_at"`
}

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userBalance struct {
	Balance   float64 `json:"balance"`
	Withdrawn float64 `json:"withdrawn"`
}

type orderWithWithdrawn struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
