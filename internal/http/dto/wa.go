package dto

type PairRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type PairResponse struct {
	PairingId   string `json:"pairing_id"`
	PairingCode string `json:"pairing_code"`
}

type SendTextRequest struct {
	From string `json:"from" binding:"required"`
	To   string `json:"to" binding:"required"`
	Text string `json:"text" binding:"required"`
}

type LogoutRequest struct {
	Phone string `json:"phone" binding:"required"`
}
