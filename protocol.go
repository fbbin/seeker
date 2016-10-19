package seeker

import "net"

type Protocol interface {
	ReadPacket(conn *net.TCPConn) (Packet, error)
}
