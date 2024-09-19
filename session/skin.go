package session

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/zeppelinmc/zeppelin/protocol/net/packet/login"
)

func getSkin(xuid string) (login.Property, error) {
	res, err := http.Get("https://api.geysermc.org/v2/skin/" + xuid)
	if err != nil {
		return login.Property{}, err
	}
	var response struct {
		Signature string `json:"signature"`
		Value     string `json:"value"`
	}
	err = json.NewDecoder(res.Body).Decode(&response)

	return login.Property{
		Name:      "textures",
		Value:     response.Value,
		Signature: response.Signature,
	}, err
}

func (session *BedrockSession) skin() protocol.Skin {
	client := session.conn.ClientData()

	var skinData, _ = base64.StdEncoding.DecodeString(client.SkinData)
	var skinResourcePatch, _ = base64.StdEncoding.DecodeString(client.SkinResourcePatch)
	var animationData, _ = base64.StdEncoding.DecodeString(client.SkinAnimationData)
	var capeData, _ = base64.StdEncoding.DecodeString(client.CapeData)

	return protocol.Skin{
		SkinID:            client.SkinID,
		SkinColour:        client.SkinColour,
		SkinImageWidth:    uint32(client.SkinImageWidth),
		SkinImageHeight:   uint32(client.SkinImageHeight),
		SkinData:          skinData,
		SkinResourcePatch: skinResourcePatch,
		AnimationData:     animationData,

		CapeImageWidth:  uint32(client.CapeImageWidth),
		CapeImageHeight: uint32(client.CapeImageHeight),
		CapeData:        capeData,
		CapeID:          client.CapeID,
		PremiumSkin:     client.PremiumSkin,
	}
}
