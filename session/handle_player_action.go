package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/zeppelinmc/zeppelin/net/metadata"
)

func handlePlayerAction(s *BedrockSession, action *packet.PlayerAction) {
	var md metadata.Metadata
	switch action.ActionType {
	case protocol.PlayerActionStartSneak:
		base := s.player.MetadataIndex(metadata.BaseIndex).(metadata.Byte)
		md = metadata.Metadata{
			metadata.BaseIndex: base | metadata.IsCrouching,
			metadata.PoseIndex: metadata.Sneaking,
		}
	case protocol.PlayerActionStopSneak:
		base := s.player.MetadataIndex(metadata.BaseIndex).(metadata.Byte)
		md = metadata.Metadata{
			metadata.BaseIndex: base &^ metadata.IsCrouching,
			metadata.PoseIndex: metadata.Standing,
		}
	case protocol.PlayerActionStartSprint:
		base := s.player.MetadataIndex(metadata.BaseIndex).(metadata.Byte)
		md = metadata.Metadata{
			metadata.BaseIndex: base | metadata.IsSprinting,
		}
	case protocol.PlayerActionStopSprint:
		base := s.player.MetadataIndex(metadata.BaseIndex).(metadata.Byte)
		md = metadata.Metadata{
			metadata.BaseIndex: base &^ metadata.IsSprinting,
		}
	}

	s.srv.Broadcast.EntityMetadata(s, md)
}
