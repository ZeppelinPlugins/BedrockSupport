package session

import (
	"aetherbedrocksupport/world"
	"net"
	"sync"

	"github.com/dynamitemc/aether/chat"
	"github.com/dynamitemc/aether/log"
	"github.com/dynamitemc/aether/net/packet/login"
	"github.com/dynamitemc/aether/net/packet/play"
	"github.com/dynamitemc/aether/server"
	"github.com/dynamitemc/aether/server/player"
	"github.com/dynamitemc/aether/server/session"
	"github.com/dynamitemc/aether/util"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func HandleNewConn(srv *server.Server, conn *minecraft.Conn) {
	id := srv.NewEntityId()

	player := player.NewPlayer(id)

	uuid, _ := uuid.Parse(conn.IdentityData().Identity)

	session := &BedrockSession{
		conn:   conn,
		player: player,
		uuid:   uuid,
		srv:    srv,
	}

	session.skinProperty, _ = getSkin(session.conn.IdentityData().XUID)

	session.sendWorldData()

	conn.StartGame(minecraft.GameData{
		WorldName:       "Server",
		EntityUniqueID:  int64(id),
		PlayerGameMode:  1,
		EntityRuntimeID: uint64(id),
		GameRules: []protocol.GameRule{
			{
				Name:                  "showcoordinates",
				CanBeModifiedByPlayer: true,
				Value:                 true,
			},
		},
	})

	srv.Broadcast.AddPlayer(session)
	srv.Broadcast.SpawnPlayer(session)

	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			srv.Broadcast.RemovePlayer(session)
			return
		}
		log.Printlnf("0x%02x %T", pk.ID(), pk)
	}
}

type BedrockSession struct {
	conn         *minecraft.Conn
	player       *player.Player
	srv          *server.Server
	uuid         uuid.UUID
	skinProperty login.Property

	spawnedEntities []int32
	spawned_ents_mu sync.Mutex
}

func (session *BedrockSession) Addr() net.Addr {
	return session.conn.RemoteAddr()
}

func (session *BedrockSession) ClientName() string {
	return "vanilla-bedrock"
}

func (session *BedrockSession) DespawnEntities(ids ...int32) error {
	for id := range ids {
		if err := session.conn.WritePacket(&packet.RemoveActor{EntityUniqueID: int64(id)}); err != nil {
			return err
		}
	}
	return nil
}

func (session *BedrockSession) Disconnect(reason chat.TextComponent) error {
	if err := session.conn.WritePacket(&packet.Disconnect{Message: reason.Text}); err != nil {
		return err
	}
	return session.conn.Close()
}

func (session *BedrockSession) EntityAnimation(entityId int32, animation byte) error {
	var anim int32
	switch animation {
	case play.AnimationSwingMainArm, play.AnimationSwingOffhand:
		anim = packet.AnimateActionSwingArm
	case play.AnimationCriticalEffect:
		anim = packet.AnimateActionCriticalHit
	case play.AnimationMagicCriticalEffect:
		anim = packet.AnimateActionMagicCriticalHit
	case play.AnimationLeaveBed:
		anim = packet.AnimateActionStopSleep
	}
	return session.conn.WritePacket(&packet.Animate{
		EntityRuntimeID: uint64(entityId),
		ActionType:      anim,
	})
}

func (session *BedrockSession) EntityMetadata(entityId int32, metadata map[byte]any) error {
	return nil
}

func (session *BedrockSession) IsSpawned(entityId int32) bool {
	session.spawned_ents_mu.Lock()
	defer session.spawned_ents_mu.Unlock()
	for _, id := range session.spawnedEntities {
		if id == entityId {
			return true
		}
	}
	return false
}

func (session *BedrockSession) Player() *player.Player {
	return session.player
}

func (session *BedrockSession) PlayerChatMessage(play.ChatMessage, session.Session, int32) error {
	return nil
}

func (session *BedrockSession) PlayerInfoRemove(uuids ...uuid.UUID) error {
	var entries = make([]protocol.PlayerListEntry, len(uuids))

	for i, uuid := range uuids {
		entries[i] = protocol.PlayerListEntry{
			UUID: uuid,
		}
	}
	return session.conn.WritePacket(&packet.PlayerList{
		ActionType: packet.PlayerListActionRemove,
		Entries:    entries,
	})
}

func (session *BedrockSession) sendWorldData() {
	viewDistance := int32(12)

	for x := -viewDistance; x < viewDistance; x++ {
		for z := -viewDistance; z < viewDistance; z++ {
			c, _ := session.srv.World.GetChunk(x, z)

			session.conn.WritePacket(&packet.LevelChunk{
				Position:      [2]int32{x, z},
				SubChunkCount: uint32(len(c.Sections)),
				RawPayload:    world.EncodeChunkData(c),
			})
		}
	}
}

func (session *BedrockSession) PlayerInfoUpdate(pk *play.PlayerInfoUpdate) error {
	/*if pk.Actions&play.ActionAddPlayer == 0 {
		return nil
	}
	var entries = make([]protocol.PlayerListEntry, 0, len(pk.Players))

	for uuid, player := range pk.Players {
		if uuid == session.uuid {
			continue
		}
		ses, ok := session.srv.Broadcast.Session(uuid)
		if !ok {
			continue
		}
		entries = append(entries, protocol.PlayerListEntry{
			UUID:           uuid,
			Username:       player.Name,
			EntityUniqueID: int64(ses.Player().EntityId()),
		})
	}

	return session.conn.WritePacket(&packet.PlayerList{
		ActionType: packet.PlayerListActionAdd,
		Entries:    entries,
	})*/
	return nil
}

func (session *BedrockSession) Properties() []login.Property {
	return []login.Property{session.skinProperty}
}

func (session *BedrockSession) SessionData() (play.PlayerSession, bool) {
	return play.PlayerSession{}, false
}

func (session *BedrockSession) SpawnPlayer(ses session.Session) error {
	x, y, z := ses.Player().Position()
	yaw, pitch := ses.Player().Rotation()
	return session.conn.WritePacket(&packet.AddPlayer{
		UUID:            ses.UUID(),
		Username:        ses.Username(),
		EntityRuntimeID: uint64(ses.Player().EntityId()),
		Position:        mgl32.Vec3{float32(x), float32(y), float32(z)},
		Pitch:           pitch,
		Yaw:             yaw,
		HeadYaw:         yaw,
	})
}

func (session *BedrockSession) SpawnEntity(ent *play.SpawnEntity) error {
	return session.conn.WritePacket(&packet.AddActor{
		EntityUniqueID:  int64(ent.EntityId),
		EntityRuntimeID: uint64(ent.EntityId),
		Position:        mgl32.Vec3{float32(ent.X), float32(ent.Y), float32(ent.Z)},
		Pitch:           util.AngleToDegrees(ent.Pitch),
		Yaw:             util.AngleToDegrees(ent.Yaw),
		BodyYaw:         util.AngleToDegrees(ent.Yaw),
		HeadYaw:         util.AngleToDegrees(ent.HeadYaw),
	})

}

func (session *BedrockSession) Teleport(x, y, z float64, yaw, pitch float32) error {
	return session.conn.WritePacket(&packet.MovePlayer{
		EntityRuntimeID: uint64(session.player.EntityId()),
		Position:        mgl32.Vec3{float32(x), float32(y), float32(z)},
		Pitch:           pitch,
		Yaw:             yaw,
		HeadYaw:         yaw,
		Mode:            packet.MoveModeTeleport,
		TeleportCause:   packet.TeleportCauseCommand,
	})
}

func (session *BedrockSession) UUID() uuid.UUID {
	return session.uuid
}

func (session *BedrockSession) UpdateEntityPosition(*play.UpdateEntityPosition) error {
	return nil
}

func (session *BedrockSession) UpdateEntityPositionRotation(*play.UpdateEntityPositionAndRotation) error {
	return nil
}

func (session *BedrockSession) UpdateEntityRotation(*play.UpdateEntityRotation) error {
	return nil
}

func (session *BedrockSession) Username() string {
	return session.conn.IdentityData().DisplayName
}

var _ session.Session = (*BedrockSession)(nil)
