package world

import (
	"bytes"
	"sync"

	"github.com/zeppelinmc/zeppelin/server/world/chunk"
)

/*func chunkBlocks(chunk *region.Chunk, sec int) []Block {
	section := chunk.Sections[sec]
	bitsPerEntry := (int32(len(section.BlockStates.Data)) * 64) / 4096
	var blocks = make([]Block, 0, 4096)

	for _, long := range section.BlockStates.Data {
		for i, pos := 0, int32(0); i < 64; i++ {
			if 64-pos < bitsPerEntry {
				break
			}
			entry := ((long << pos) >> pos) & int64(bitsPerEntry)

			blocks = append(blocks, findBlock(section.BlockStates.Palette[entry].Name))

			pos += bitsPerEntry
		}
	}

	return blocks
}*/

var chunkDataPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(nil) },
}

func EncodeChunkData(chunk *chunk.Chunk) []byte {
	w := chunkDataPool.Get().(*bytes.Buffer)
	w.Reset()
	defer chunkDataPool.Put(w)

	for i := 0; i < 24; i++ {
		blocksPerWord := byte(1)
		w.WriteByte(1 | (blocksPerWord << 1))

		for j := 0; j < 1024; j++ {
			w.Write([]byte{
				255, 255, 255, 255,
			})
		}
		w.WriteByte(1)
		w.WriteByte(7)
	}
	/*for i := range chunk.Sections {
		blocksPerWord := byte(16)
		w.WriteByte(1 | (blocksPerWord << 1))

		var palette = make(map[int]int)

		blocks := chunkBlocks(chunk, i)

		var plI int
		for _, block := range blocks {
			if _, ok := palette[block.RuntimeId]; !ok {
				palette[block.RuntimeId] = plI
				plI++
			}
		}

		for blI := 0; blI < len(blocks); blI += 2 {
			block0 := blocks[blI]
			block1 := blocks[blI+1]

			plIn0 := palette[block0.RuntimeId]
			plIn1 := palette[block1.RuntimeId]

			entry := uint32(plIn0) | uint32(plIn1<<32)
			binary.Write(w, binary.LittleEndian, entry)
		}
		protocol.WriteVarint32(w, int32(len(palette)))
		for _, e := range palette {
			protocol.WriteVarint32(w, int32(e))
		}
	}*/
	return w.Bytes()
}

func findBlock(name string) Block {
	for _, block := range blockTable {
		if block.Name == name {
			return block
		}
	}
	return blockTable[0]
}
