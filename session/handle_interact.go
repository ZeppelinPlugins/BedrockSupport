package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func handleInteract(session *BedrockSession, interact *packet.Interact) {
	if interact.ActionType == packet.InteractActionOpenInventory && !session.inInv.Get() {
		x, y, z := session.player.Position()

		session.conn.WritePacket(&packet.ContainerOpen{
			WindowID:                0,
			ContainerType:           0xff,
			ContainerEntityUniqueID: -1,
			ContainerPosition: protocol.BlockPos{
				int32(x),
				int32(y),
				int32(z),
			},
		})
		session.inInv.Set(true)
	}
}
