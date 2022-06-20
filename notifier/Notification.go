package notifier

type Notification struct {
	Topic string
	Data  map[string]interface{}
}