package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/zeppelinmc/zeppelin/net/metadata"
)

func javaMDtoBedrockMD(meta metadata.Metadata) protocol.EntityMetadata {
	md := protocol.NewEntityMetadata()
	base, _ := meta[metadata.BaseIndex].(metadata.Byte)
	if base&metadata.IsOnFire != 0 {
		md.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagOnFire)
	}
	if base&metadata.IsCrouching != 0 {
		md.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSneaking)
	}
	if base&metadata.IsSprinting != 0 {
		md.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSprinting)
	}
	if base&metadata.IsSwimming != 0 {
		md.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSwimming)
	}

	return md
}
