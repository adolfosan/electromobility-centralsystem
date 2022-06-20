package actions

import (
	"central_system/common"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-playground/validator"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

var reservationId int = 0

type ReservationProfileActions struct {
	centralSystem ocpp16.CentralSystem
}

func convertToReserveNowRequest(payload []byte) (*reservation.ReserveNowRequest, *common.Error) {
	var data map[string]interface{} = make(map[string]interface{})

	json.Unmarshal(payload, &data)
	//fmt.Printf("%+v  %\n", data)
	//verificar formato de fecha
	/*expiryDate, exists := data["expiryDate"]
	if !exists {
		fmt.Println(" convertToReserveNowRequest ->No existe la propiedad expiryDate")
		return nil, &common.Error{
			Code:    "command.reserve.now.payload.not.valid",
			Message: fmt.Sprintf("Campos no válidos para realizar reservación en el Punto de Carga: %v", " propiedad expiryDate no existe"),
		}
	}*/

	//fmt.Printf("%+v", data)
	connectorId, _ := strconv.ParseInt(fmt.Sprint(data["connectorId"]), 10, 32)

	request := &reservation.ReserveNowRequest{
		IdTag:         "34556",
		ExpiryDate:    types.NewDateTime(time.Now().Add(5 * time.Minute)),
		ReservationId: reservationId + 1,
		ConnectorId:   int(connectorId),
	}

	// validar idTag
	if idTag, existIdTag := data["idTag"]; existIdTag {
		request.IdTag = fmt.Sprint(idTag)
		if len(request.IdTag) == 0 {
			return nil, &common.Error{
				Code:    "command.reserve.now.payload.not.valid",
				Message: fmt.Sprintf("El identificador no puede ser vacio "),
			}
		}

	} else {
		return nil, &common.Error{
			Code:    "command.reserve.now.payload.not.valid",
			Message: fmt.Sprintf("Campos no válidos para realizar reservación en el Punto de Carga: %v", " propiedad idTag no existe"),
		}
	}

	// validar expiryDate

	if expiryDate, existExpiryDate := data["expiryDate"]; existExpiryDate {

		var s string = fmt.Sprint(expiryDate)
		intVar, err := strconv.ParseFloat(s, 64)
		if err == nil {
			secondsEpoch := int64(intVar)
			request.ExpiryDate = types.NewDateTime(time.Unix(secondsEpoch, 0))
		} else {

			return nil, &common.Error{
				Code:    "command.reserve.now.payload.not.valid",
				Message: fmt.Sprintf("Campos no válidos para realizar reservación en el Punto de Carga: %v", " propiedad expiryDate no es un número"),
			}
		}
	} else {
		return nil, &common.Error{
			Code:    "command.reserve.now.payload.not.valid",
			Message: fmt.Sprintf("Campos no válidos para realizar reservación en el Punto de Carga: %v", " propiedad expiryDate no existe"),
		}
	}

	fmt.Printf(" +%v \n", request)
	/*if secondsEpoch, err := strconv.ParseInt(expiryDate.(string), 10, 64); err != nil {
		return nil, &common.Error{
			Code:    "command.reserve.now.payload.not.valid",
			Message: fmt.Sprintf("Campos no válidos para realizar reservación en el Punto de Carga: %v", " propiedad expiryDate no es un número"),
		}
	} else {
		fmt.Println(time.Now())
		fmt.Println(time.Unix(secondsEpoch, 0))
		fmt.Println(time.Unix(secondsEpoch, 0).Local())
		fmt.Println(time.Unix(secondsEpoch, 0).Unix())

		request.ExpiryDate = types.NewDateTime(time.Unix(secondsEpoch, 0).Local())
		request.IdTag = data["idTag"].(string)

		var Validator = validator.New()
		json.Unmarshal(payload, request)
		err := Validator.Struct(request)

		fmt.Printf("%+v \n", request)

		if err != nil {
			fmt.Printf("%+v", err)
			return nil, &common.Error{
				Code:    "command.reserve.now.payload.not.valid",
				Message: "Campos no válidos para realizar reservación en el Punto de Carga.",
			}

		}

	}*/

	return request, nil
}

func InitializeReservationProfileActions(centralSystem ocpp16.CentralSystem) ReservationProfileActions {

	return ReservationProfileActions{
		centralSystem: centralSystem,
	}
}

func (this *ReservationProfileActions) ReserveNow(chargePointID string, payload []byte, responseChannel chan common.Response) {

	/*responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: ReserveNow no esta implementada"),
		},
	}*/

	var response common.Response

	request, ts := convertToReserveNowRequest(payload)
	if ts != nil {
		fmt.Println("convertToReserveNowRequest")
		responseChannel <- common.Response{
			Err: ts,
		}
		return
	}
	/*fmt.Println("ReserveNow")
	request := &reservation.ReserveNowRequest{
		IdTag:         "34556",
		ExpiryDate:    types.NewDateTime(time.Now().Add(5 * time.Minute)),
		ReservationId: reservationId + 1,
		ConnectorId:   1,
	}*/

	//fmt.Printf(" +%v \n", request)

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
					request.IdTag, request.ExpiryDate.Local())
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
	e := this.centralSystem.ReserveNow(chargePointID, cb, request.ConnectorId, request.ExpiryDate, request.IdTag, reservationId)
	if e != nil {
		logDefault(chargePointID, reservation.ReserveNowFeatureName).Errorf("couldn't send message: %v", e)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		reservationId = reservationId - 1
		responseChannel <- response
	}
}

func (this *ReservationProfileActions) CancelReservation(chargePointID string, payload []byte, responseChannel chan common.Response) {
	fmt.Println("ESTOY AQUI")
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

	e := this.centralSystem.CancelReservation(chargePointID, cb, request.ReservationId)
	if e != nil {
		logDefault(chargePointID, reservation.CancelReservationFeatureName).Errorf("couldn't send message: %v", e)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}
