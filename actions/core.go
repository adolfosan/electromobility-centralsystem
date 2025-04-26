package actions

import (
	"central_system/common"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-playground/validator"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/sirupsen/logrus"
)

var chargingProfileId int = 0

func logDefault(chargePointId string, feature string) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{"client": chargePointId, "message": feature})
}

type Function func(string, []byte, chan common.Response)

type CoreProfileActions struct {
	centralSystem ocpp16.CentralSystem
}

func InitializeCoreProfileActions(centralSystem ocpp16.CentralSystem) CoreProfileActions {

	return CoreProfileActions{
		centralSystem: centralSystem,
	}
}

func (cp *CoreProfileActions) Reset(chargePointID string, payload []byte, responseChannel chan common.Response) {

	var response common.Response

	var data map[string]interface{} = make(map[string]interface{})

	errUnMarshal := json.Unmarshal(payload, &data)

	if errUnMarshal != nil {
		response.Err = &common.Error{
			Code:    "command.remote.start.transaction",
			Message: "Conversion a json no valida",
		}
		responseChannel <- response
		return
	}

	var resetType core.ResetType = core.ResetTypeSoft

	if fmt.Sprintf("%v", data["type"]) == "Hard" {
		resetType = core.ResetTypeHard
	}

	cb := func(confirmation *core.ResetConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.ResetFeatureName).Errorf("error on request: %v", err)
		} else {
			var (
				payload map[string]interface{} = make(map[string]interface{})
				status  core.ResetStatus       = confirmation.Status
				message string                 = ""
			)
			switch status {
			case core.ResetStatusAccepted:
				//logDefault(chargePointID, confirmation.GetFeatureName()).Infof("reset %v canceled successfully", request.Type)
				message = fmt.Sprintf("Se ha aceptado el reinicio por el modo: %v", resetType)
			case core.ResetStatusRejected:
				//logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't cancel reservation %v", request.Type)
				message = " No se ha aceptado el reinicio."
			}
			payload["status"] = status
			payload["message"] = message
			response.Payload = payload
		}

		responseChannel <- response
	}

	e := cp.centralSystem.Reset(chargePointID, cb, resetType)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func (cp *CoreProfileActions) GetConfiguration(chargePointID string, payload []byte, responseChannel chan common.Response) {

	var response common.Response

	var Validator = validator.New()
	request := &core.GetConfigurationRequest{}
	json.Unmarshal(payload, request)

	err := Validator.Struct(request)

	if err != nil {
		response.Err = &common.Error{
			Code:    "command.get.configuration.payload.not.valid",
			Message: "Campos no válidos para obtener la configuración del Punto de Carga.",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *core.GetConfigurationConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.GetConfigurationFeatureName).Errorf("error on request: %v", err)
		} else {
			var payload map[string]interface{} = make(map[string]interface{})

			for _, configurationKey := range confirmation.ConfigurationKey {
				payload[configurationKey.Key] = struct {
					Readonly bool        `json:"readonly"`
					Value    interface{} `json:"value"`
				}{
					configurationKey.Readonly,
					*configurationKey.Value,
				}
			}
			response.Payload = payload
		}
		responseChannel <- response
	}

	e := cp.centralSystem.GetConfiguration(chargePointID, cb, request.Key)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func (cp *CoreProfileActions) ChangeConfiguration(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response

	var Validator = validator.New()
	request := &core.ChangeConfigurationRequest{}

	json.Unmarshal(payload, request)
	err := Validator.Struct(request)

	if err != nil {
		response.Err = &common.Error{
			Code:    "command.change.configuration.payload.not.valid",
			Message: "Campos no válidos para cambiar un elemento de la configuración del Punto de Carga.",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *core.ChangeConfigurationConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.ChangeConfigurationFeatureName).Errorf("error on request: %v", err)
		} else if confirmation.Status == core.ConfigurationStatusNotSupported {
			response.Err = &common.Error{
				Code:    "command.change.configuration.key.unsupported",
				Message: fmt.Sprintf("La variable %v no existe en la configuracion del punto de carga: %v", request.Key, chargePointID),
			}
		} else if confirmation.Status == core.ConfigurationStatusRejected {
			response.Err = &common.Error{
				Code:    "command.change.configuration.readonly",
				Message: fmt.Sprintf("La variable (%v) es solo de lectura", request.Key),
			}
		} else if confirmation.Status == core.ConfigurationStatusRebootRequired {
			response.Payload = fmt.Sprintf("La variable %v ha sido actualizada, pero su cambio estará disponible después de reiniciar el punto de carga.", request.Key)
		} else {
			response.Payload = fmt.Sprintf("La variable %v ha sido actualizada.", request.Key)
		}
		responseChannel <- response
	}

	e := cp.centralSystem.ChangeConfiguration(chargePointID, cb, request.Key, request.Value)
	if e != nil {
		//logDefault(chargePointID, localauth.GetLocalListVersionFeatureName).Errorf("couldn't send message: %v", e)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func (cp *CoreProfileActions) ChangeAvailability(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response

	request := &core.ChangeAvailabilityRequest{}
	// DANDO ERROR EN LA VALIDACION OJO!!!!!!!!
	json.Unmarshal(payload, request)
	//fmt.Printf("%+v", request)
	//log.Info(request)

	/*var Validator = validator.New()
	err := Validator.Struct(request)
	if err != nil {
		response.Err = &common.Error{
			Code:    "command.change.availability.payload.not.valid",
			Message: "Campos no válidos para cambiar el estado operativo del Punto de Carga.",
		}
		responseChannel <- response
		return
	}*/

	cb := func(confirmation *core.ChangeAvailabilityConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.GetConfigurationFeatureName).Errorf("error on request: %v", err)
		} else {
			var (
				payload map[string]interface{}  = make(map[string]interface{})
				status  core.AvailabilityStatus = confirmation.Status
				message string                  = ""
			)

			switch status {
			case core.AvailabilityStatusAccepted:
				message = fmt.Sprintf("El conector %v ha sido actualizado al estado: %v", request.ConnectorId, request.Type)
			case core.AvailabilityStatusRejected:
				message = fmt.Sprintf("El conector %v ha rechazado el estado: %v", request.ConnectorId, request.Type)
			case core.AvailabilityStatusScheduled:
				message = fmt.Sprintf("El conector %v ha sido programado para cambiar al estado: %v , cuando haya finalizado con sus transaccion(es) ", request.ConnectorId, request.Type)
			}

			payload["status"] = status
			payload["message"] = message
			response.Payload = payload
		}
		responseChannel <- response
	}

	e := cp.centralSystem.ChangeAvailability(chargePointID, cb, request.ConnectorId, request.Type)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func (cp *CoreProfileActions) RemoteStartTransaction(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response
	var data map[string]interface{} = make(map[string]interface{})

	errUnMarshal := json.Unmarshal(payload, &data)

	if errUnMarshal != nil {
		response.Err = &common.Error{
			Code:    "command.remote.start.transaction",
			Message: "Campos no válidos para iniciar una transaccion remota. 1",
		}
		responseChannel <- response
		return
	}

	var idTag string = ""

	var connectorId *int = new(int)
	*connectorId = -1

	var duration *int = new(int)
	*duration = -1

	if _, ok := data["idTag"]; !ok {
		response.Err = &common.Error{
			Code:    "command.remote.start.transaction",
			Message: "IdTag is required",
		}
		responseChannel <- response
		return
	}

	idTag = fmt.Sprint(data["idTag"])

	if _, ok := data["connectorId"]; ok {
		ci, errInt := strconv.ParseInt(fmt.Sprint(data["connectorId"]), 10, 32)

		if errInt != nil {
			response.Err = &common.Error{
				Code:    "command.remote.start.transaction",
				Message: "connectorId must be a integer",
			}
			responseChannel <- response
			return
		} else if ci < 1 {
			response.Err = &common.Error{
				Code:    "command.remote.start.transaction",
				Message: "connectorId must be g(0)",
			}
			responseChannel <- response
			return
		}
		*connectorId = int(ci)
	}

	if _, ok := data["duration"]; ok {
		d, errInt := strconv.ParseInt(fmt.Sprint(data["duration"]), 10, 32)

		if errInt != nil {
			response.Err = &common.Error{
				Code:    "command.remote.start.transaction",
				Message: "duration must be a integer",
			}
			responseChannel <- response
			return
		} else if d < 1 {
			response.Err = &common.Error{
				Code:    "command.remote.start.transaction",
				Message: "duration must be g(0)",
			}
			responseChannel <- response
			return
		}
		fmt.Println(d)
		*duration = int(d)
	}

	cb := func(confirmation *core.RemoteStartTransactionConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.RemoteStartTransactionFeatureName).Errorf("error on request: %v", err)
		} else {
			var payload map[string]interface{} = make(map[string]interface{})

			payload["status"] = confirmation.Status
			response.Payload = payload

			if confirmation.Status != types.RemoteStartStopStatusAccepted {
				chargingProfileId = chargingProfileId - 1
			}
		}

		responseChannel <- response

	}

	e := cp.centralSystem.RemoteStartTransaction(chargePointID, cb, idTag, func(req *core.RemoteStartTransactionRequest) {
		chargingProfileId = chargingProfileId + 1
		var periods []types.ChargingSchedulePeriod = []types.ChargingSchedulePeriod{
			{
				StartPeriod: 0,
				Limit:       18.3,
			},
		}
		req.IdTag = idTag
		req.ConnectorId = connectorId
		req.ChargingProfile = &types.ChargingProfile{
			ChargingProfileId:      chargingProfileId,
			StackLevel:             1,
			ChargingProfilePurpose: types.ChargingProfilePurposeTxProfile,
			ChargingProfileKind:    types.ChargingProfileKindRecurring,
			ChargingSchedule: &types.ChargingSchedule{
				Duration:               duration,
				ChargingRateUnit:       types.ChargingRateUnitWatts,
				ChargingSchedulePeriod: periods,
			},
		}
		fmt.Printf("IdTag %v \n", req.IdTag)
		fmt.Printf("ConnectorId %v \n", *req.ConnectorId)
		fmt.Printf("ChargingProfile => id  %v \n", req.ChargingProfile.ChargingProfileId)
		fmt.Printf("ChargingProfile => stackLevel  %v \n", req.ChargingProfile.StackLevel)
		fmt.Printf("ChargingProfile => Schedule => duration  %v \n", *req.ChargingProfile.ChargingSchedule.Duration)
		fmt.Printf("ChargingProfile => Schedule => rateUnit  %v \n", req.ChargingProfile.ChargingSchedule.ChargingRateUnit)

	})

	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}

	/*responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: RemoteStartTransaction no esta implementada"),
		},
	}*/
}

func (cp *CoreProfileActions) RemoteStopTransaction(chargePointID string, payload []byte, responseChannel chan common.Response) {

	var response common.Response

	var data map[string]interface{} = make(map[string]interface{})
	errUnMarshal := json.Unmarshal(payload, &data)

	if errUnMarshal != nil {
		response.Err = &common.Error{
			Code:    "command.remote.stop.transaction",
			Message: "Conversion no json no valida",
		}
		responseChannel <- response
		return
	}

	transactionId, errInt := strconv.ParseInt(fmt.Sprint(data["transactionId"]), 10, 32)

	if errInt != nil {
		response.Err = &common.Error{
			Code:    "command.remote.stop.transaction",
			Message: "transactionId must be a integer",
		}
		responseChannel <- response
		return
	} else if transactionId < 0 {
		response.Err = &common.Error{
			Code:    "command.remote.stop.transaction",
			Message: "transactionId must be gte(0)",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *core.RemoteStopTransactionConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.RemoteStopTransactionFeatureName).Errorf("error on request: %v", err)
		} else {
			var payload map[string]interface{} = make(map[string]interface{})

			payload["status"] = confirmation.Status
			response.Payload = payload
		}

		responseChannel <- response
	}

	e := cp.centralSystem.RemoteStopTransaction(chargePointID, cb, int(transactionId))

	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func (cp *CoreProfileActions) UnlockConnector(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response
	var data map[string]interface{} = make(map[string]interface{})
	errUnMarshal := json.Unmarshal(payload, &data)

	if errUnMarshal != nil {
		response.Err = &common.Error{
			Code:    "command.unlock.connector",
			Message: "Conversion no json no valida",
		}
		responseChannel <- response
		return
	}

	var connectorId int
	if _, ok := data["connectorId"]; ok {
		ci, errInt := strconv.ParseInt(fmt.Sprint(data["connectorId"]), 10, 32)
		if errInt != nil {
			response.Err = &common.Error{
				Code:    "command.unlock.connector",
				Message: "connectorId must be an integer",
			}
			responseChannel <- response
			return
		}
		connectorId = int(ci)
	} else {
		response.Err = &common.Error{
			Code:    "command.unlock.connector",
			Message: "connectorId is required",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *core.UnlockConnectorConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.UnlockConnectorFeatureName).Errorf("error on request: %v", err)
		} else {
			var payload map[string]interface{} = make(map[string]interface{})

			payload["status"] = confirmation.Status
			response.Payload = payload
		}

		responseChannel <- response
	}

	e := cp.centralSystem.UnlockConnector(chargePointID, cb, connectorId)

	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}

	/*responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: UnlockConnector no esta implementada"),
		},
	}*/
}

func (cp *CoreProfileActions) ClearCache(chargePointID string, payload []byte, responseChannel chan common.Response) {

	var response common.Response

	cb := func(confirmation *core.ClearCacheConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, core.ClearCacheFeatureName).Errorf("error on request: %v", err)
		} else {
			var payload map[string]interface{} = make(map[string]interface{})

			payload["status"] = confirmation.Status
			response.Payload = payload
		}

		responseChannel <- response
	}

	e := cp.centralSystem.ClearCache(chargePointID, cb)

	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}

	/*responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: ClearCache no esta implementada"),
		},
	}*/
}
