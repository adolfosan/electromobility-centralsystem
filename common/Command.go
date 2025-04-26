package common

type Command struct {
	Action        string      `json:"action" validate:"required"`
	ChargePointId string      `json:"chargePointId" validate:"required"`
	Payload       interface{} `json:"payload"`
}
