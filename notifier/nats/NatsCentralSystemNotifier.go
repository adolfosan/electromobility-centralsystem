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

func (this *natsCentralSystemNotifier) SetTimeout(timeout time.Duration) {
	this.timeout = timeout

}
func (this natsCentralSystemNotifier) Timeout() time.Duration {
	return this.timeout
}

func (this *natsCentralSystemNotifier) AddHandler(action string, fn Function) {
	this.handlers[action] = fn
}

func (this *natsCentralSystemNotifier) SetChannel(notification chan notifier.Notification) {
	this.notification = notification
}

func (this natsCentralSystemNotifier) notificationFromCentralSystem() {
	for {
		n := <-this.notification
		bt, err := json.Marshal(n.Data)

		if err != nil {
			log.Error(err)
		} else {
			this.connection.Publish(n.Topic, bt)
		}
	}
}

// funcion asociada al patron request/reply en Nats
func (this natsCentralSystemNotifier) requestHandler() {

	var Validator = validator.New()

	this.connection.Subscribe("request", func(m *nats.Msg) {

		var command common.Command
		json.Unmarshal(m.Data, &command)

		validate := Validator.Struct(&command)

		if validate != nil {
			bt, _ := json.Marshal(common.Response{
				Err: &common.Error{
					Code:    "command.format.not.valid",
					Message: "El comando no es válido",
				},
			})
			m.Respond(bt)
			return

		}

		_, exists := this.handlers[command.Action]

		if !exists {
			bt, _ := json.Marshal(common.Response{
				Err: &common.Error{
					Code:    "command.action.not.found",
					Message: fmt.Sprintf("No existe la acción \"%v\"", command.Action),
				},
			})
			m.Respond(bt)
			return
		}

		var responseChannel chan common.Response = make(chan common.Response)
		payload, _ := json.Marshal(command.Payload)

		var fn Function = this.handlers[command.Action]

		go fn(command.ChargePointId, payload, responseChannel)

		select {
		case response := <-responseChannel:
			//log.Info("RESULT!!!!")
			bt, _ := json.Marshal(response)
			m.Respond(bt)
			break
		case <-time.After(this.timeout):
			//log.Error("TIMEOUT!!!!")
			bt, _ := json.Marshal(common.Response{
				Err: &common.Error{
					Code:    "request.timeout",
					Message: "Ha caducado el tiempo de respuesta de la solicitud",
				},
			})
			m.Respond(bt)
			break
		}
	})
}

func (this *natsCentralSystemNotifier) Start() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		//log.Fatal( err)
		log.Fatal(err)
	}
	this.connection = nc
	go this.notificationFromCentralSystem()
	go this.requestHandler()
}

func (this *natsCentralSystemNotifier) Stop() {
	if this.connection != nil {
		this.connection.Close()
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
