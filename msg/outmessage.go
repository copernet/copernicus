package msg

type OutMessage struct {
	Message Message
	Done    chan<- struct{}
}
