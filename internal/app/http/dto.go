package http

type PairCodeRequest struct {
	Phone string `json:"phone"`
}

type PairCodeResponse struct {
	Status      string `json:"status"`
	PairingCode string `json:"pairing_code,omitempty"`
	JID         string `json:"jid,omitempty"`
}

type MeResponse struct {
	Status   string `json:"status"`
	Id       string `json:"id"`
	LID      string `json:"lid,omitempty"`
	JID      string `json:"jid,omitempty"`
	PushName string `json:"pushName"`
}

type ClientsResponse struct {
	Count   int      `json:"count"`
	Clients []string `json:"clients"`
}
