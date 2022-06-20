package actions

import (
	"central_system/common"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/lorenzodonini/ocpp-go/ocpp"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
)

type LocalAuthProfileActions struct {
	centralSystem ocpp16.CentralSystem
}

func InitializeLocalAuthProfileActions(centralSystem ocpp16.CentralSystem) LocalAuthProfileActions {

	return LocalAuthProfileActions{
		centralSystem: centralSystem,
	}
}

func (this *LocalAuthProfileActions) SendLocalListVersion(chargePointID string, payload []byte, responseChannel chan common.Response) {
	/*responseChannel <- common.Response{
		Err: &common.Error{
			Code:    "not.implemented",
			Message: fmt.Sprintf("La funcionalidad: SendLocalListVersion no esta implementada"),
		},
	}*/

	var response common.Response

	var data map[string]interface{} = make(map[string]interface{})

	errUnMarshal := json.Unmarshal(payload, &data)

	if errUnMarshal != nil {
		response.Err = &common.Error{
			Code:    "command.send.local.list.version",
			Message: "Conversion a json no valida",
		}
		responseChannel <- response
		return
	}

	listVersion, errInt := strconv.ParseInt(fmt.Sprint(data["listVersion"]), 10, 32)

	if errInt != nil {
		response.Err = &common.Error{
			Code:    "command.send.local.list.version",
			Message: "listVersion must be a integer",
		}
		responseChannel <- response
		return
	}

	cb := func(confirmation *localauth.SendLocalListConfirmation, err error) {
		if err != nil {
			response.Err = &common.Error{
				Code:    "command.send.local.list.version.request.error",
				Message: fmt.Sprintf("No se pudo enviar la lista local al Punto de Carga por: %v", err),
			}
		} else {
			response.Payload = confirmation.Status
		}
		responseChannel <- response
	}

	err := this.centralSystem.SendLocalList(chargePointID, cb, int(listVersion), localauth.UpdateTypeFull,
		func(request *localauth.SendLocalListRequest) {
			arrStr := strings.Split(fmt.Sprintf("%v", data["localAuthorizationList"]), ",")
			var list []localauth.AuthorizationData = []localauth.AuthorizationData{}
			for _, id := range arrStr {
				list = append(list,
					localauth.AuthorizationData{
						IdTag: id,
					},
				)
			}
			request.LocalAuthorizationList = list
		})

	if err != nil {
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}

func (this *LocalAuthProfileActions) GetLocalListVersion(chargePointID string, payload []byte, responseChannel chan common.Response) {
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
	err := this.centralSystem.SendRequestAsync(chargePointID, request, generalCB)
	if err != nil {
		//log.Printf("error sending message: %v", err)
		response.Err = &common.Error{
			Code:    "command.message.not.send",
			Message: fmt.Sprintf("No se pudo enviar el comando al Punto de Carga: %v", chargePointID),
		}
		responseChannel <- response
	}
}
