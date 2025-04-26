package main 

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

type Connector struct {
	status             core.ChargePointStatus
	currentTransaction int
}


func (this *Connector) hasTransactionInProgress() bool {
	return this.currentTransaction >= 0
}