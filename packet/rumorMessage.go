package packet

type RumorMessage struct {
	Origin string //message's original sender
	ID     uint32 //monotonically increasing sequence number assigned by the original sender
	Text   string //content of the message
}
