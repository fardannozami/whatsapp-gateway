package http

type PairCodeRequest struct {
	Phone string `json:"phone"`
}

type PairCodeResponse struct {
	Status      string `json:"status"`
	PairingCode string `json:"pairing_code,omitempty"`
	JID         string `json:"jid,omitempty"`
}

type SendTextRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type SendTextResponse struct {
	Status    string `json:"status"`
	MessageID string `json:"message_id,omitempty"`
}

type PairStreamResponse struct {
	Status      string `json:"status"`
	PairingCode string `json:"pairing_code,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	RetryIn     int    `json:"retry_in,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

type SessionItemResponse struct {
	Session  string `json:"session"`
	ID       string `json:"id,omitempty"`
	PushName string `json:"pushName,omitempty"`
	Status   string `json:"status"`
}

type SessionsStreamResponse struct {
	Status   string                `json:"status"`
	Sessions []SessionItemResponse `json:"sessions,omitempty"`
	Detail   string                `json:"detail,omitempty"`
}

type DeleteSessionResponse struct {
	Status string `json:"status"`
}

type StopSessionResponse struct {
	Status string `json:"status"`
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
