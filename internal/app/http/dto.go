package http

type PairCodeRequest struct {
	Phone string `json:"phone"`
}

type PairCodeResponse struct {
	Status      string `json:"status"`
	PairingCode string `json:"pairing_code,omitempty"`
	JID         string `json:"jid,omitempty"`
}

type ClientsResponse struct {
	Count   int      `json:"count"`
	Clients []string `json:"clients"`
}
