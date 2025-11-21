package handlers

type userOrdersResponse struct {
	OrderNumber string   `json:"number" db:"order_num"`
	Status      string   `json:"status" db:"status"`
	Accrual     *float64 `json:"accrual,omitempty" db:"accrual"`
	CreatedAt   string   `json:"uploaded_at" db:"created_at"`
}

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
