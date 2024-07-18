package world

import (
	_ "embed"
	"encoding/json"
)

type Block struct {
	Data      int    `json:"data"`
	ID        int    `json:"id"`
	Name      string `json:"name"`
	RuntimeId int    `json:"runtimeID"`
}

//go:embed runtimeid_table.json
var runtime_id_table []byte

var blockTable []Block

func init() {
	json.Unmarshal(runtime_id_table, &blockTable)
}
