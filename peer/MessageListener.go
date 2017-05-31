package peer

type MessageListener struct {
	OnGetAddr func()
}