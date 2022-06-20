package main

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
)

type ChargePoint struct {
	status            core.ChargePointStatus
	diagnosticsStatus firmware.DiagnosticsStatus
	firmwareStatus    firmware.FirmwareStatus
	connectors        map[int]*Connector // No assumptions about the # of connectors
	transactions      map[int]*Transaction
	errorCode         core.ChargePointErrorCode
}

func (this *ChargePoint) getConnector(id int) *Connector {
	ci, ok := this.connectors[id]
	if !ok {
		ci = &Connector{currentTransaction: -1}
		this.connectors[id] = ci
	}
	return ci
}
