package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"

	"central_system/actions"
	notifier "central_system/notifier/nats"
	"time"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"
)

const (
	defaultListenPort          = 8887
	defaultHeartbeatInterval   = 600
	envVarServerPort           = "SERVER_LISTEN_PORT"
	envVarTls                  = "TLS_ENABLED"
	envVarCaCertificate        = "CA_CERTIFICATE_PATH"
	envVarServerCertificate    = "SERVER_CERTIFICATE_PATH"
	envVarServerCertificateKey = "SERVER_CERTIFICATE_KEY_PATH"
)

const (
	GET_CONFIGURATION        = "get.configuration"
	CHANGE_CONFIGURATION     = "change.configuration"
	CHANGE_AVAILABILITY      = "change.avalability"
	GET_LOCAL_LIST_VERSION   = "get.local.list.version"
	SEND_LOCAL_LIST_VERSION  = "send.local.list.version"
	RESERVE_NOW              = "reserve.now"
	CANCEL_RESERVATION       = "cancel.reservation"
	RESET                    = "reset"
	REMOTE_START_TRANSACTION = "remote.start.transaction"
	REMOTE_STOP_TRANSACTION  = "remote.stop.transaction"
	UNLOCK_CONNECTOR         = "unlock.connector"
	CLEAR_CACHE              = "clear.cache"
	CLEAR_CHARGING_PROFILE   = "clear.charging.profile"
	GET_COMPOSITE_SCHEDULE   = "get.composite.schedule"
	SET_CHARGING_PROFILE     = "set.charging.profile"
)

var log *logrus.Logger
var centralSystem ocpp16.CentralSystem

func setupCentralSystem() ocpp16.CentralSystem {
	return ocpp16.NewCentralSystem(nil, nil)
}

func setupTlsCentralSystem() ocpp16.CentralSystem {
	var certPool *x509.CertPool
	// Load CA certificates
	caCertificate, ok := os.LookupEnv(envVarCaCertificate)
	if !ok {
		log.Infof("no %v found, using system CA pool", envVarCaCertificate)
		systemPool, err := x509.SystemCertPool()
		if err != nil {
			log.Fatalf("couldn't get system CA pool: %v", err)
		}
		certPool = systemPool
	} else {
		certPool = x509.NewCertPool()
		data, err := ioutil.ReadFile(caCertificate)
		if err != nil {
			log.Fatalf("couldn't read CA certificate from %v: %v", caCertificate, err)
		}
		ok = certPool.AppendCertsFromPEM(data)
		if !ok {
			log.Fatalf("couldn't read CA certificate from %v", caCertificate)
		}
	}
	certificate, ok := os.LookupEnv(envVarServerCertificate)
	if !ok {
		log.Fatalf("no required %v found", envVarServerCertificate)
	}
	key, ok := os.LookupEnv(envVarServerCertificateKey)
	if !ok {
		log.Fatalf("no required %v found", envVarServerCertificateKey)
	}
	server := ws.NewTLSServer(certificate, key, &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  certPool,
	})
	return ocpp16.NewCentralSystem(nil, server)
}

// Start function
func main() {
	centralSystem = setupCentralSystem()

	// Support callbacks for all OCPP 1.6 profiles
	//handler := &CentralSystemHandler{chargePoints: map[string]*ChargePointState{}}
	csHandler := NewCentralSystemHandler()
	centralSystem.SetCoreHandler(csHandler)
	/*centralSystem.SetLocalAuthListHandler(csHandler)
	centralSystem.SetFirmwareManagementHandler(csHandler)
	centralSystem.SetReservationHandler(csHandler)
	centralSystem.SetRemoteTriggerHandler(csHandler)
	centralSystem.SetSmartChargingHandler(csHandler)*/

	ocppj.SetLogger(log)
	ocppj.SetMessageValidation(false)

	centralSystem.SetNewChargePointHandler(func(chargePoint ocpp16.ChargePointConnection) {
		csHandler.chargePoints[chargePoint.ID()] = &ChargePoint{connectors: map[int]*Connector{}, transactions: map[int]*Transaction{}}
		log.WithField("client", chargePoint.ID()).Info("new charge point connected")
		//go example(chargePoint.ID(), handler)
	})

	centralSystem.SetChargePointDisconnectedHandler(func(chargePoint ocpp16.ChargePointConnection) {
		log.WithField("client", chargePoint.ID()).Info("charge point disconnected")
		delete(csHandler.chargePoints, chargePoint.ID())
	})

	natsNotifier := notifier.New()
	natsNotifier.SetChannel(csHandler.NotificationChannel())
	natsNotifier.SetTimeout(3 * time.Minute)
	log.Printf("Esperar respuesta de las solicitudes: %v", natsNotifier.Timeout().String())

	//Usando las funciones del Callbacks.go

	coreProfileActions := actions.InitializeCoreProfileActions(centralSystem)
	//localAuthProfileActions := actions.InitializeLocalAuthProfileActions(centralSystem)
	//reservationProfileActions := actions.InitializeReservationProfileActions(centralSystem)
	//smartChargingProfilesActions := actions.InitializeSmartChargingProfileActions(centralSystem)

	natsNotifier.AddHandler(RESET, coreProfileActions.Reset)
	natsNotifier.AddHandler(GET_CONFIGURATION, coreProfileActions.GetConfiguration)
	natsNotifier.AddHandler(CHANGE_CONFIGURATION, coreProfileActions.ChangeConfiguration)
	natsNotifier.AddHandler(CHANGE_AVAILABILITY, coreProfileActions.ChangeAvailability)
	natsNotifier.AddHandler(REMOTE_START_TRANSACTION, coreProfileActions.RemoteStartTransaction)
	natsNotifier.AddHandler(REMOTE_STOP_TRANSACTION, coreProfileActions.RemoteStopTransaction)
	natsNotifier.AddHandler(UNLOCK_CONNECTOR, coreProfileActions.UnlockConnector)
	natsNotifier.AddHandler(CLEAR_CACHE, coreProfileActions.ClearCache)

	/*natsNotifier.AddHandler(SEND_LOCAL_LIST_VERSION, localAuthProfileActions.SendLocalListVersion)
	natsNotifier.AddHandler(GET_LOCAL_LIST_VERSION, localAuthProfileActions.GetLocalListVersion)

	natsNotifier.AddHandler(RESERVE_NOW, reservationProfileActions.ReserveNow)
	natsNotifier.AddHandler(CANCEL_RESERVATION, reservationProfileActions.CancelReservation)

	natsNotifier.AddHandler(CLEAR_CHARGING_PROFILE, smartChargingProfilesActions.ClearChargingProfile)
	natsNotifier.AddHandler(GET_COMPOSITE_SCHEDULE, smartChargingProfilesActions.GetCompositeSchedule)
	natsNotifier.AddHandler(SET_CHARGING_PROFILE, smartChargingProfilesActions.SetChargingProfile)*/

	natsNotifier.Start()
	defer natsNotifier.Stop()

	// Run central system
	log.Infof("starting central system on port %v", 8887)
	centralSystem.Start(8887, "/{ws}")

	log.Info("stopped central system")
}

func init() {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	// Set this to DebugLevel if you want to retrieve verbose logs from the ocppj and websocket layers
	log.SetLevel(logrus.InfoLevel)
}
