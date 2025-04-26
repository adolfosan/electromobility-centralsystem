package actions

import (
	"central_system/common"
	"encoding/json"
	"fmt"
	"log"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
)

type SmartChargingProfileActions struct {
	centralSystem ocpp16.CentralSystem
}

func InitializeSmartChargingProfileActions(centralSystem ocpp16.CentralSystem) SmartChargingProfileActions {

	return SmartChargingProfileActions{
		centralSystem: centralSystem,
	}
}

func (this *SmartChargingProfileActions) ClearChargingProfile(chargePointID string, payload []byte, responseChannel chan common.Response) {
	responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: ClearChargingProfile no esta implementada."),
		},
	}
}

func (this *SmartChargingProfileActions) GetCompositeSchedule(chargePointID string, payload []byte, responseChannel chan common.Response) {
	responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: GetCompositeSchedule no esta implementada."),
		},
	}
}

func (this *SmartChargingProfileActions) SetChargingProfile(chargePointID string, payload []byte, responseChannel chan common.Response) {

	/*responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: SetChargingProfile no esta implementada."),
		},
	}*/

	var response common.Response

	var req smartcharging.SetChargingProfileRequest
	err := json.Unmarshal(payload, &req)

	if err != nil {
		log.Fatal(err)
	}

	cb := func(confirmation *smartcharging.SetChargingProfileConfirmation, err error) {
		if err != nil {
			logDefault(chargePointID, smartcharging.SetChargingProfileFeatureName).Errorf("error on request: %v", err)
		} else {
			var (
				payload map[string]interface{}              = make(map[string]interface{})
				status  smartcharging.ChargingProfileStatus = confirmation.Status
				message string                              = ""
			)

			switch status {
			case smartcharging.ChargingProfileStatusAccepted:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("Se ha aceptado el perfil de carga")
				message = fmt.Sprintf(" Se ha aceptado el perfil de carga")
			case smartcharging.ChargingProfileStatusRejected:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("No se ha aceptado el perfil de carga")
				message = fmt.Sprintf(" No se ha aceptado el perfil de carga")
			/*case smartcharging.ChargingProfileStatusNotImplemented:
				logDefault(chargePointID, confirmation.GetFeatureName()).Infof("La solicitud no es soportada por el cargador")
				message = fmt.Sprintf(" La solicitud no es soportada por el cargador")*/
			}

			payload["status"] = status
			payload["message"] = message
			response.Payload = payload
		}
		responseChannel <- response
	}

	e := this.centralSystem.SetChargingProfile(chargePointID, cb, req.ConnectorId, req.ChargingProfile)

	if e != nil {
		logDefault(chargePointID, smartcharging.SetChargingProfileFeatureName).Errorf("couldn't send message: %v", e)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}
