package main

import (
	"central_system/common"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lorenzodonini/ocpp-go/ocpp"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

var reservationId int = 0

//FUNCION PLANTILLA
/*func Template(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response
	response.Payload = " Template not implemented"

	responseChannel <- response
}*/

func Reset(chargePointID string, payload []byte, responseChannel chan common.Response) {
	/*var response common.Response
	response.Payload = " Reset not implemented"

	responseChannel <- response*/

	var response common.Response

	request := &core.ResetRequest{
		Type: core.ResetTypeSoft, //se borra cuando se realice la validacion
	}

	json.Unmarshal(payload, request)
	// DANDO ERROR EN LA VALIDACION OJO!!!!!!!!
	/*var Validator = validator.New()
	err := Validator.Struct(request)

	if err != nil {
		response.Err = &common.Error{
			Code:    "command.reset.payload.not.valid",
			Message: "Campos no válidos para reiniciar el Punto de Carga.",
		}
		responseChannel <- response
		return
	}*/

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
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("reservation %v canceled successfully", request.Type)
				message = fmt.Sprintf("Se ha aceptado el reinicio del Punto de Carga: por el modo: %v", request.Type)
			case core.ResetStatusRejected:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't cancel reservation %v", request.Type)
				message = fmt.Sprintf(" No se ha aceptado el reinicio del Punto de Carga")
			}
			payload["status"] = status
			payload["message"] = message
			response.Payload = payload
		}

		responseChannel <- response
	}

	e := centralSystem.Reset(chargePointID, cb, request.Type)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func CancelReservation(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response

	var Validator = validator.New()
	request := &reservation.CancelReservationRequest{}

	json.Unmarshal(payload, request)
	err := Validator.Struct(request)

	if err != nil {
		response.Err = &common.Error{
			Code:    "command.cancel.reservation.payload.not.valid",
			Message: "Campos no válidos para cancelar la reservación en el Punto de Carga.",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *reservation.CancelReservationConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, reservation.CancelReservationFeatureName).Errorf("error on request: %v", err)
		} else {
			var (
				payload map[string]interface{}              = make(map[string]interface{})
				status  reservation.CancelReservationStatus = confirmation.Status
				message string                              = ""
			)
			switch status {
			case reservation.CancelReservationStatusAccepted:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("reservation %v canceled successfully", request.ReservationId)
				message = fmt.Sprintf(" La reservación %v ha sido cancelada", request.ReservationId)
			default:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't cancel reservation %v", request.ReservationId)
				message = fmt.Sprintf(" La reservación %v no ha sido cancelada", request.ReservationId)
			}
			payload["status"] = status
			payload["message"] = message
			response.Payload = payload
		}

		responseChannel <- response
	}

	e := centralSystem.CancelReservation(chargePointID, cb, request.ReservationId)
	if e != nil {
		//logDefault(chargePointID, localauth.GetLocalListVersionFeatureName).Errorf("couldn't send message: %v", e)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func ReserveNow(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response

	var Validator = validator.New()
	request := &reservation.ReserveNowRequest{
		ExpiryDate:    types.NewDateTime(time.Now().Add(5 * time.Minute)),
		ReservationId: reservationId + 1,
	}

	json.Unmarshal(payload, request)
	err := Validator.Struct(request)

	if err != nil {
		response.Err = &common.Error{
			Code:    "command.reserve.now.payload.not.valid",
			Message: "Campos no válidos para realizar reservación en el Punto de Carga.",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *reservation.ReserveNowConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, reservation.ReserveNowFeatureName).Errorf("error on request: %v", err)
			reservationId = reservationId - 1
		} else {
			var (
				payload map[string]interface{}        = make(map[string]interface{})
				status  reservation.ReservationStatus = confirmation.Status
				message string                        = ""
			)

			switch status {
			case reservation.ReservationStatusAccepted:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("connector %v reserved for client %v until %v (reservation ID %d)",
					request.ConnectorId, request.IdTag, request.ExpiryDate.FormatTimestamp(), request.ReservationId)
				message = fmt.Sprintf(" El conector %v ha sido reservado por el cliente %v hasta %v", request.ConnectorId,
					request.IdTag, request.ExpiryDate.FormatTimestamp())
				payload["reservationId"] = reservationId
			case reservation.ReservationStatusFaulted:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't reserve connector %v: %v", request.ConnectorId, status)
				message = fmt.Sprintf(" No se ha podido realizar la reservación en el conector %v por estar en estado de Falla.", request.ConnectorId)
				reservationId = reservationId - 1
			case reservation.ReservationStatusOccupied:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't reserve connector %v: %v", request.ConnectorId, status)
				message = fmt.Sprintf(" No se ha podido realizar la reservación en el conector %v por estar ocupado.", request.ConnectorId)
				reservationId = reservationId - 1
			case reservation.ReservationStatusRejected:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't reserve connector %v: %v", request.ConnectorId, status)
				message = fmt.Sprintf(" No se ha podido realizar la reservación en el conector %v porque el Punto de Carga no permite realizar reservaciones", request.ConnectorId)
				reservationId = reservationId - 1
			case reservation.ReservationStatusUnavailable:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't reserve connector %v: %v", request.ConnectorId, status)
				message = fmt.Sprintf(" No se ha podido realizar la reservación en el conector %v por estar deshabilitado.", request.ConnectorId)
				reservationId = reservationId - 1
			}

			payload["status"] = status
			payload["message"] = message
			response.Payload = payload
		}
		responseChannel <- response
	}

	reservationId = reservationId + 1
	e := centralSystem.ReserveNow(chargePointID, cb, request.ConnectorId, request.ExpiryDate, request.IdTag, reservationId)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		reservationId = reservationId - 1
		responseChannel <- response
	}
}

func GetConfiguration(chargePointID string, payload []byte, responseChannel chan common.Response) {

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

	e := centralSystem.GetConfiguration(chargePointID, cb, request.Key)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func ChangeConfiguration(chargePointID string, payload []byte, responseChannel chan common.Response) {
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
			//logDefault(chargePointID, core.ChangeConfigurationFeatureName).Errorf("error on request: %v", err)
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

	e := centralSystem.ChangeConfiguration(chargePointID, cb, request.Key, request.Value)
	if e != nil {
		//logDefault(chargePointID, localauth.GetLocalListVersionFeatureName).Errorf("couldn't send message: %v", e)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func SendLocalListVersion(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response

	request := &localauth.SendLocalListRequest{
		/*ListVersion: 1,
		UpdateType:  localauth.UpdateTypeDifferential,*/
	}

	// DANDO ERROR EN LA VALIDACION OJO!!!!!!!!
	json.Unmarshal(payload, request)

	//log.Info(request)

	/*var Validator = validator.New()
	validate := Validator.Struct(request)

	if validate != nil {
		response.Err = &common.Error{
			Code:    "command.send.local.list.payload.not.valid",
			Message: "Campos no válidos para actualizar la lista local del Punto de Carga.",
		}
		responseChannel <- response
		return
	}*/

	cb := func(confirmation *localauth.SendLocalListConfirmation, err error) {
		if err != nil {
			//log.Info("SendLocalListConfirmation")
			response.Err = &common.Error{
				Code:    "command.send.local.list.version.request.error",
				Message: fmt.Sprintf("No se pudo enviar la lista local al Punto de Carga por: %v", err),
			}
		} else {
			response.Payload = confirmation.Status
		}
		responseChannel <- response
	}
	generalCB := func(confirmation ocpp.Response, protoError error) {
		if confirmation != nil {
			cb(confirmation.(*localauth.SendLocalListConfirmation), protoError)
		} else {
			cb(nil, protoError)
		}
	}
	err := centralSystem.SendRequestAsync(chargePointID, request, generalCB)
	if err != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func ChangeAvailability(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response

	request := &core.ChangeAvailabilityRequest{}
	// DANDO ERROR EN LA VALIDACION OJO!!!!!!!!
	json.Unmarshal(payload, request)
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

	e := centralSystem.ChangeAvailability(chargePointID, cb, request.ConnectorId, request.Type)
	if e != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func GetLocalListVersion(chargePointID string, payload []byte, responseChannel chan common.Response) {
	var response common.Response
	request := &localauth.GetLocalListVersionRequest{}

	generalCB := func(confirmation ocpp.Response, protoError error) {
		if confirmation != nil {
			if protoError != nil {
				response.Err = &common.Error{
					Code:    "command.get.local.list.version.request.error",
					Message: fmt.Sprintf("No se pudo obtener la version de la lista local por: %v", protoError),
				}
			} else {
				getLocalListVersionConfirmation := confirmation.(*localauth.GetLocalListVersionConfirmation)
				response.Payload = getLocalListVersionConfirmation.ListVersion
			}
		} else {
			//log.Error(protoError)
			response.Err = &common.Error{
				Code:    "command.get.local.list.version.request.error",
				Message: fmt.Sprintf("No se pudo obtener la versión de la lista local por: %v", protoError),
			}
		}
		responseChannel <- response
	}
	err := centralSystem.SendRequestAsync(chargePointID, request, generalCB)
	if err != nil {
		//log.Printf("error sending message: %v", err)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}
