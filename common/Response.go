package common

type Response struct {
	Payload interface{} `json:"payload"`
	Err     *Error      `json:"error"`
}
