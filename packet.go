package seeker

type Packet interface {
	Serialize() []byte
}
