package main

import (
	"zeppelinbedrocksupport/session"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/zeppelinmc/zeppelin/server"
	"github.com/zeppelinmc/zeppelin/util/log"
)

const (
	VERSION = "1.0"
)

var ZeppelinPluginExport = server.Plugin{
	Identifier: "ZeppelinBedrockSupport",
	OnLoad: func(p *server.Plugin) {
		log.Infolnf("Zeppelin Bedrock Support version %s", VERSION)

		srvConf := p.Server().Properties()

		cfg := minecraft.ListenConfig{
			StatusProvider: minecraft.NewStatusProvider("Zeppelin", srvConf.MOTD),
		}

		listener, err := cfg.Listen("raknet", ":19132")
		if err != nil {
			log.Errorlnf("Zeppelin Bedrock Support: error listening: %v", err)
			return
		}
		log.Infolnf("Zeppelin Bedrock Support: listening on %s", listener.Addr())

		go func() {
			for {
				select {
				case <-close:
					return
				default:
					c, err := listener.Accept()
					if err != nil {
						log.Errorlnf("Zeppelin Bedrock Support: error listening: %v", err)
						return
					}
					conn := c.(*minecraft.Conn)
					go session.HandleNewConn(p.Server(), conn)
				}
			}
		}()
	},
}

var close = make(chan struct{})
