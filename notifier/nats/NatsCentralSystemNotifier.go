package notifier

import (
	"central_system/common"
	"central_system/notifier"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
}

//type Function func( string, []byte, *common.Response)

type Function func(string, []byte, chan common.Response)

type natsCentralSystemNotifier struct {
	notification chan notifier.Notification //canal por el cual se envia el resultado de las operaciones del CP al CS
	connection   *nats.Conn                 //conexion a Nats
	handlers     map[string]Function        //mapa de funciones
	timeout      time.Duration              //tiempo de espera de las solicitudes
}

func (ncs *natsCentralSystemNotifier) SetTimeout(timeout time.Duration) {
	ncs.timeout = timeout

}
func (ncs natsCentralSystemNotifier) Timeout() time.Duration {
	return ncs.timeout
}

func (ncs *natsCentralSystemNotifier) AddHandler(action string, fn Function) {
	ncs.handlers[action] = fn
}

func (ncs *natsCentralSystemNotifier) SetChannel(notification chan notifier.Notification) {
	ncs.notification = notification
}

func (ncs natsCentralSystemNotifier) notificationFromCentralSystem() {
	for {
		n := <-ncs.notification
		bt, err := json.Marshal(n.Data)

		if err != nil {
			log.Error(err)
		} else {
			ncs.connection.Publish(n.Topic, bt)
		}
	}
}

// funcion asociada al patron request/reply en Nats
func (n *natsCentralSystemNotifier) requestHandler() {

	var Validator = validator.New()

	n.connection.Subscribe("request", func(m *nats.Msg) {

		var command common.Command
		json.Unmarshal(m.Data, &command)
		log.Printf("RequestHandler, %+v", string(m.Data))
		validate := Validator.Struct(&command)

		if validate != nil {
			bt, _ := json.Marshal(common.Response{
				Err: &common.Error{
					Code:    "command.format.not.valid",
					Message: "El comando no es válido",
				},
			})
			log.Errorf("%v", bt)
			m.Respond(bt)
			return

		}

		_, exists := n.handlers[command.Action]

		if !exists {
			bt, _ := json.Marshal(common.Response{
				Err: &common.Error{
					Code:    "command.action.not.found",
					Message: fmt.Sprintf("No existe la acción \"%v\"", command.Action),
				},
			})
			log.Errorf("%v", bt)
			m.Respond(bt)
			return
		}

		var responseChannel chan common.Response = make(chan common.Response)
		payload, _ := json.Marshal(command.Payload)

		var fn Function = n.handlers[command.Action]

		go fn(command.ChargePointId, payload, responseChannel)

		select {
		case response := <-responseChannel:
			bt, _ := json.Marshal(response)
			log.Printf("RequestHandler => Response, %v", string(bt))
			m.Respond(bt)
			break
		case <-time.After(n.timeout):
			bt, _ := json.Marshal(common.Response{
				Err: &common.Error{
					Code:    "request.timeout",
					Message: "Ha caducado el tiempo de respuesta de la solicitud",
				},
			})
			log.Errorf("%v", bt)
			m.Respond(bt)
			break
		}
	})
}

func (ncs *natsCentralSystemNotifier) Start() {

	nc, err := nats.Connect(nats.DefaultURL)
	//nc, err := nats.Connect("tls://connect.ngs.global", nats.UserCredentials("./config/nats.creds"))
	if err != nil {
		log.Fatal(err)
	}
	ncs.connection = nc
	go ncs.notificationFromCentralSystem()
	go ncs.requestHandler()
}

func (ncs *natsCentralSystemNotifier) Stop() {
	if ncs.connection != nil {
		ncs.connection.Close()
		log.Info("NatsStopped")
	}
}

func New() *natsCentralSystemNotifier {
	return &natsCentralSystemNotifier{
		notification: nil,
		connection:   nil,
		handlers:     make(map[string]Function),
		timeout:      30 * time.Second,
	}
}
