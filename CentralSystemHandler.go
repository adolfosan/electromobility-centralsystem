package main

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"

	"central_system/notifier"
	"encoding/json"
)

var (
	nextTransactionId = 0
)

// TransactionInfo contains info about a transaction
type Transaction struct {
	id          int
	startTime   *types.DateTime
	endTime     *types.DateTime
	meterStart  int
	meterStop   int
	connectorId int
	idTag       string
}
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "Active"
	SessionStatusCompleted SessionStatus = "Completed"
	SessionStatusCanceled  SessionStatus = "Canceled"
	SessionStatusPending   SessionStatus = "Pending"
)

type Session struct {
	Transaction
	user string
}

func (ti *Transaction) hasTransactionEnded() bool {
	return ti.endTime != nil && !ti.endTime.IsZero()
}

// CentralSystemHandler contains some simple state that a central system may want to keep.
// In production this will typically be replaced by database/API calls.
type CentralSystemHandler struct {
	chargePoints map[string]*ChargePoint
	notification chan notifier.Notification
}

// ------------- Core profile callbacks -------------

func (handler *CentralSystemHandler) OnAuthorize(chargePointId string, request *core.AuthorizeRequest) (confirmation *core.AuthorizeConfirmation, err error) {

	//logDefault(chargePointId, request.GetFeatureName()).Infof("client authorized")

	return core.NewAuthorizationConfirmation(types.NewIdTagInfo(types.AuthorizationStatusAccepted)), nil
}

func (handler *CentralSystemHandler) OnBootNotification(chargePointId string, request *core.BootNotificationRequest) (confirmation *core.BootNotificationConfirmation, err error) {
	//logDefault(chargePointId, request.GetFeatureName()).Infof("boot confirmed")
	//fmt.Println(request)
	var data = make(map[string]interface{})
	data["chargePointId"] = chargePointId

	bt, _ := json.Marshal(request)
	json.Unmarshal(bt, &data)

	handler.notification <- notifier.Notification{
		Topic: "boot.notification",
		Data:  data,
	}

	return core.NewBootNotificationConfirmation(types.NewDateTime(time.Now()), defaultHeartbeatInterval, core.RegistrationStatusAccepted), nil
}

func (handler *CentralSystemHandler) OnDataTransfer(chargePointId string, request *core.DataTransferRequest) (confirmation *core.DataTransferConfirmation, err error) {
	//logDefault(chargePointId, request.GetFeatureName()).Infof("received data %d", request.Data)

	var m = make(map[string]interface{})
	m["chargePointId"] = chargePointId

	handler.notification <- notifier.Notification{
		Topic: "data_transfer",
	}

	return core.NewDataTransferConfirmation(core.DataTransferStatusAccepted), nil
}

func (handler *CentralSystemHandler) OnHeartbeat(chargePointId string, request *core.HeartbeatRequest) (confirmation *core.HeartbeatConfirmation, err error) {
	//logDefault(chargePointId, request.GetFeatureName()).Infof("heartbeat handled")
	var currentTime *types.DateTime = types.NewDateTime(time.Now())

	var m = make(map[string]interface{})
	m["chargePointId"] = chargePointId
	m["currentTime"] = currentTime

	handler.notification <- notifier.Notification{
		Topic: "heartbeat",
	}

	return core.NewHeartbeatConfirmation(types.NewDateTime(time.Now())), nil
}

func (handler *CentralSystemHandler) OnMeterValues(chargePointId string, request *core.MeterValuesRequest) (confirmation *core.MeterValuesConfirmation, err error) {
	//logDefault(chargePointId, request.GetFeatureName()).Infof("received meter values for connector %v. Meter values:\n", request.ConnectorId)

	var data = make(map[string]interface{})
	data["chargePointId"] = chargePointId

	bt, _ := json.Marshal(request)
	json.Unmarshal(bt, &data)

	handler.notification <- notifier.Notification{
		Topic: "meter.values",
		Data:  data,
	}

	/*for _, mv := range request.MeterValue {
		logDefault(chargePointId, request.GetFeatureName()).Printf("%v", mv)
	}*/
	return core.NewMeterValuesConfirmation(), nil
}

func (handler *CentralSystemHandler) OnStatusNotification(chargePointId string, request *core.StatusNotificationRequest) (confirmation *core.StatusNotificationConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	info.errorCode = request.ErrorCode
	if request.ConnectorId > 0 {
		connectorInfo := info.getConnector(request.ConnectorId)
		connectorInfo.status = request.Status
	} else {
		info.status = request.Status
	}

	var data = make(map[string]interface{})
	data["chargePointId"] = chargePointId

	bt, _ := json.Marshal(request)
	json.Unmarshal(bt, &data)

	handler.notification <- notifier.Notification{
		Topic: "status.notification",
		Data:  data,
	}

	return core.NewStatusNotificationConfirmation(), nil
}

func (handler *CentralSystemHandler) OnStartTransaction(chargePointId string, request *core.StartTransactionRequest) (confirmation *core.StartTransactionConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]

	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	connector := info.getConnector(request.ConnectorId)
	if connector.currentTransaction >= 0 {
		return nil, fmt.Errorf("connector %v is currently busy with another transaction", request.ConnectorId)
	}
	transaction := &Transaction{}
	transaction.idTag = request.IdTag
	transaction.connectorId = request.ConnectorId
	transaction.meterStart = request.MeterStart
	transaction.startTime = request.Timestamp
	transaction.id = nextTransactionId
	nextTransactionId += 1
	connector.currentTransaction = transaction.id
	info.transactions[transaction.id] = transaction

	var data = make(map[string]interface{})
	data["chargePointId"] = chargePointId
	data["transactionId"] = transaction.id

	bt, _ := json.Marshal(request)
	json.Unmarshal(bt, &data)

	handler.notification <- notifier.Notification{
		Topic: "start.transaction",
		Data:  data,
	}

	return core.NewStartTransactionConfirmation(types.NewIdTagInfo(types.AuthorizationStatusAccepted), transaction.id), nil
}

func (handler *CentralSystemHandler) OnStopTransaction(chargePointId string, request *core.StopTransactionRequest) (confirmation *core.StopTransactionConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]

	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	transaction, ok := info.transactions[request.TransactionId]
	if ok {
		connector := info.getConnector(transaction.connectorId)
		connector.currentTransaction = -1
		transaction.endTime = request.Timestamp
		transaction.meterStop = request.MeterStop

		var data = make(map[string]interface{})
		data["chargePointId"] = chargePointId

		bt, _ := json.Marshal(request)
		json.Unmarshal(bt, &data)

		handler.notification <- notifier.Notification{
			Topic: "stop.transaction",
			Data:  data,
		}
	}

	return core.NewStopTransactionConfirmation(), nil
}

// ------------- Firmware management profile callbacks -------------

func (handler *CentralSystemHandler) OnDiagnosticsStatusNotification(chargePointId string, request *firmware.DiagnosticsStatusNotificationRequest) (confirmation *firmware.DiagnosticsStatusNotificationConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	handler.notification <- notifier.Notification{
		Topic: "diagnostic.status.notification",
	}
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	info.diagnosticsStatus = request.Status
	//logDefault(chargePointId, request.GetFeatureName()).Infof("updated diagnostics status to %v", request.Status)
	return firmware.NewDiagnosticsStatusNotificationConfirmation(), nil
}

func (handler *CentralSystemHandler) OnFirmwareStatusNotification(chargePointId string, request *firmware.FirmwareStatusNotificationRequest) (confirmation *firmware.FirmwareStatusNotificationConfirmation, err error) {
	info, ok := handler.chargePoints[chargePointId]
	handler.notification <- notifier.Notification{
		Topic: "firmware_status_notification",
	}
	if !ok {
		return nil, fmt.Errorf("unknown charge point %v", chargePointId)
	}
	info.firmwareStatus = request.Status
	//logDefault(chargePointId, request.GetFeatureName()).Infof("updated firmware status to %v", request.Status)
	return &firmware.FirmwareStatusNotificationConfirmation{}, nil
}

// No callbacks for Local Auth management, Reservation, Remote trigger or Smart Charging profile on central system

// Utility functions

func logDefault(chargePointId string, feature string) *logrus.Entry {
	return log.WithFields(logrus.Fields{"client": chargePointId, "message": feature})
}

func NewCentralSystemHandler() *CentralSystemHandler {
	return &CentralSystemHandler{
		chargePoints: map[string]*ChargePoint{},
		notification: make(chan notifier.Notification),
	}
}

func (handler *CentralSystemHandler) NotificationChannel() chan notifier.Notification {
	return handler.notification
}

func (handler CentralSystemHandler) GetChargePoint(id string) (*ChargePoint, bool) {
	cp, exist := handler.chargePoints[id]
	if !exist {
		return cp, true
	}
	return nil, false
}
