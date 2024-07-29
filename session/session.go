package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"net"
	"net/http"
	"sync"
	"zeppelinbedrocksupport/world"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/zeppelinmc/zeppelin/log"
	"github.com/zeppelinmc/zeppelin/net/metadata"
	"github.com/zeppelinmc/zeppelin/net/packet/login"
	"github.com/zeppelinmc/zeppelin/net/packet/play"
	"github.com/zeppelinmc/zeppelin/server"
	"github.com/zeppelinmc/zeppelin/server/config"
	"github.com/zeppelinmc/zeppelin/server/entity"
	"github.com/zeppelinmc/zeppelin/server/player"
	"github.com/zeppelinmc/zeppelin/server/session"
	"github.com/zeppelinmc/zeppelin/server/world/region"
	"github.com/zeppelinmc/zeppelin/text"
	"github.com/zeppelinmc/zeppelin/util"
)

func HandleNewConn(srv *server.Server, conn *minecraft.Conn) {
	id := srv.NewEntityId()

	uuid, _ := uuid.Parse(conn.IdentityData().Identity)

	player := player.New(id, srv.World.NewPlayerData(uuid))

	session := &BedrockSession{
		conn:   conn,
		player: player,
		uuid:   uuid,
		srv:    srv,
	}

	session.skinProperty, _ = getSkin(session.conn.IdentityData().XUID)

	x, y, z := player.Position()
	yaw, pitch := player.Rotation()

	//session.sendWorldData()

	conn.StartGame(minecraft.GameData{
		WorldName:       "Server",
		EntityUniqueID:  int64(id),
		PlayerGameMode:  int32(player.GameMode()),
		EntityRuntimeID: uint64(id),
		WorldSeed:       int64(srv.World.Data.WorldGenSettings.Seed),
		Hardcore:        srv.World.Data.Hardcore,
		PlayerPosition:  mgl32.Vec3{float32(x), float32(y), float32(z)},
		Yaw:             yaw,
		Pitch:           pitch,
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
		p, err := conn.ReadPacket()
		if err != nil {
			srv.Broadcast.RemovePlayer(session)
			return
		}
		switch pk := p.(type) {
		case *packet.Text:
			srv.Broadcast.DisguisedChatMessage(session, text.TextComponent{Text: pk.Message})
		case *packet.MovePlayer:
			x, y, z := float64(pk.Position.X()), float64(pk.Position.Y()), float64(pk.Position.Z())
			yaw, pitch := pk.Yaw, pk.Pitch
			srv.Broadcast.BroadcastPlayerMovement(session, x, y, z, yaw, pitch)
			session.player.SetPosition(x, y, z)
			session.player.SetRotation(yaw, pitch)
		case *packet.PlayerAction:
			handlePlayerAction(session, pk)
		case *packet.Animate:
			handleAnimate(session, pk)
		case *packet.CommandRequest:
			session.srv.CommandManager.Call(pk.CommandLine[1:], session)
		default:
			log.Printlnf("0x%02x %T", p.ID(), p)
		}
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

func (session *BedrockSession) Dimension() *region.Dimension {
	return session.srv.World.Dimension(session.player.Dimension())
}

func (session *BedrockSession) ClientName() string {
	return "vanilla-bedrock"
}

func (session *BedrockSession) Config() config.ServerConfig {
	return session.srv.Config()
}

func (session *BedrockSession) DespawnEntities(ids ...int32) error {
	for id := range ids {
		if err := session.conn.WritePacket(&packet.RemoveActor{EntityUniqueID: int64(id)}); err != nil {
			return err
		}
	}
	return nil
}

func (session *BedrockSession) Disconnect(reason text.TextComponent) error {
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

func (session *BedrockSession) EntityMetadata(entityId int32, meta metadata.Metadata) error {
	md := javaMDtoBedrockMD(meta)

	return session.conn.WritePacket(&packet.SetActorData{
		EntityRuntimeID: uint64(entityId),
		EntityMetadata:  md,
	})
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

func (session *BedrockSession) PlayerChatMessage(msg play.ChatMessage, sender session.Session, chatType string, index int32, prevMsgs []play.PreviousMessage) error {
	return session.DisguisedChatMessage(text.TextComponent{Text: msg.Message}, sender, chatType)
}

func (session *BedrockSession) DisguisedChatMessage(message text.TextComponent, sender session.Session, chatType string) error {
	return session.conn.WritePacket(&packet.Text{
		TextType:   packet.TextTypeChat,
		SourceName: sender.Username(),
		Message:    message.Text,
	})
}

func (session *BedrockSession) SystemMessage(message text.TextComponent) error {
	return session.conn.WritePacket(&packet.Text{
		TextType: packet.TextTypeRaw,
		Message:  text.Marshal(message, '§'),
	})
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

	//dim := session.Dimension()

	x, _, z := session.player.Position()

	chunkX, chunkZ := int32(math.Floor(x/16)), int32(math.Floor(z/16))

	for x := chunkX - viewDistance; x < chunkX+viewDistance; x++ {
		for z := chunkZ - viewDistance; z < chunkZ+viewDistance; z++ {
			//c, _ := dim.GetChunk(x, z)

			session.conn.WritePacket(&packet.LevelChunk{
				Position:      [2]int32{x, z},
				SubChunkCount: 24,
				RawPayload:    world.EncodeChunkData(nil),
			})
		}
	}
}

func (session *BedrockSession) PlayerInfoUpdate(pk *play.PlayerInfoUpdate) error {
	if pk.Actions&play.ActionAddPlayer == 0 {
		return nil
	}
	var entries = make([]protocol.PlayerListEntry, 0, len(pk.Players))

	for uuid, player := range pk.Players {
		if uuid == session.uuid {
			continue
		}
		ses, ok := session.srv.Broadcast.AsyncSession(uuid)
		if !ok {
			continue
		}
		var skin protocol.Skin
		txt, err := ses.Textures()
		if err == nil {
			skin, _ = texturesToSkin(txt)
		}
		entries = append(entries, protocol.PlayerListEntry{
			UUID:           uuid,
			Username:       player.Name,
			EntityUniqueID: int64(ses.Player().EntityId()),
			Skin:           skin,
		})
	}

	return session.conn.WritePacket(&packet.PlayerList{
		ActionType: packet.PlayerListActionAdd,
		Entries:    entries,
	})
}

func (session *BedrockSession) Properties() []login.Property {
	return []login.Property{session.skinProperty}
}

func texturesToSkin(textures login.Textures) (protocol.Skin, error) {
	var skin protocol.Skin
	skinData, err := http.Get(textures.Textures.Skin.URL)
	if err != nil {
		return skin, err
	}
	skinImage, err := png.Decode(skinData.Body)
	if err != nil {
		return skin, err
	}
	img, ok := skinImage.(*image.NRGBA)
	if !ok {
		return skin, fmt.Errorf("bad skin image data")
	}

	w, h := skinImage.Bounds().Dx(), skinImage.Bounds().Dy()
	skin.SkinImageWidth = uint32(w)
	skin.SkinImageHeight = uint32(h)
	skin.SkinData = img.Pix

	return skin, nil
}

func (session *BedrockSession) Textures() (login.Textures, error) {
	var textures login.Textures
	property := session.skinProperty.Value
	data, err := base64.StdEncoding.DecodeString(property)
	if err != nil {
		return textures, err
	}
	err = json.Unmarshal(data, &textures)
	return textures, err
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

func (session *BedrockSession) SpawnEntity(ent entity.Entity) error {
	x, y, z := ent.Position()
	yaw, pitch := ent.Rotation()
	id := ent.EntityId()
	return session.conn.WritePacket(&packet.AddActor{
		EntityUniqueID:  int64(id),
		EntityRuntimeID: uint64(id),
		Position:        mgl32.Vec3{float32(x), float32(y), float32(z)},
		Pitch:           pitch,
		Yaw:             yaw,
		BodyYaw:         yaw,
		HeadYaw:         yaw,
	})

}

func (session *BedrockSession) SynchronizePosition(x, y, z float64, yaw, pitch float32) error {
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

func (session *BedrockSession) UpdateEntityPosition(entity entity.Entity, pk *play.UpdateEntityPosition) error {
	oldX, oldY, oldZ := entity.Position()
	yaw, pitch := entity.Rotation()

	newX := (float64(pk.DeltaX) + (oldX * 4096)) / 4096
	newY := (float64(pk.DeltaZ) + (oldY * 4096)) / 4096
	newZ := (float64(pk.DeltaZ) + (oldZ * 4096)) / 4096

	return session.conn.WritePacket(&packet.MovePlayer{
		EntityRuntimeID: uint64(entity.EntityId()),
		Yaw:             yaw,
		Pitch:           pitch,
		HeadYaw:         yaw,
		OnGround:        pk.OnGround,
		Position:        mgl32.Vec3{float32(newX), float32(newY), float32(newZ)},
	})
}

func (session *BedrockSession) UpdateEntityPositionRotation(entity entity.Entity, pk *play.UpdateEntityPositionAndRotation) error {
	oldX, oldY, oldZ := entity.Position()

	newX := (float64(pk.DeltaX) + (oldX * 4096)) / 4096
	newY := (float64(pk.DeltaZ) + (oldY * 4096)) / 4096
	newZ := (float64(pk.DeltaZ) + (oldZ * 4096)) / 4096

	return session.conn.WritePacket(&packet.MovePlayer{
		EntityRuntimeID: uint64(entity.EntityId()),
		Yaw:             util.AngleToDegrees(pk.Yaw),
		Pitch:           util.AngleToDegrees(pk.Pitch),
		HeadYaw:         util.AngleToDegrees(pk.Yaw),
		OnGround:        pk.OnGround,
		Position:        mgl32.Vec3{float32(newX), float32(newY), float32(newZ)},
	})
}

func (session *BedrockSession) UpdateEntityRotation(entity entity.Entity, pk *play.UpdateEntityRotation) error {
	x, y, z := entity.Position()
	return session.conn.WritePacket(&packet.MovePlayer{
		EntityRuntimeID: uint64(entity.EntityId()),
		Yaw:             util.AngleToDegrees(pk.Yaw),
		Pitch:           util.AngleToDegrees(pk.Pitch),
		HeadYaw:         util.AngleToDegrees(pk.Yaw),
		OnGround:        pk.OnGround,
		Position:        mgl32.Vec3{float32(x), float32(y), float32(z)},
	})
}

func (session *BedrockSession) Username() string {
	return session.conn.IdentityData().DisplayName
}

func (session *BedrockSession) UpdateTime(int64, int64) error {
	return nil
}

var _ session.Session = (*BedrockSession)(nil)
