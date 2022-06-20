package notifier

type CentralSystemNotifierInterface interface {
	Synchronize()
	Start()
	Stop()
}

/*func NewCentralSystemNotifier( notification chan Notification)* centralSystemNotifier {
	return &centralSystemNotifier {
		notification: notification,
	}
}*/