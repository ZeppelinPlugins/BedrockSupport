package session

import (
	"encoding/json"
	"net/http"

	"github.com/zeppelinmc/zeppelin/net/packet/login"
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
