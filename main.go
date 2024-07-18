package main

import (
	"aetherbedrocksupport/session"

	"github.com/dynamitemc/aether/log"
	"github.com/dynamitemc/aether/server"
	"github.com/sandertv/gophertunnel/minecraft"
)

const (
	VERSION = "1.0"
)

type Plugin struct {
	srv    *server.Server
	closed bool
}

func (p *Plugin) OnLoad(srv *server.Server) {
	log.Infolnf("Aether Bedrock Support version %s", VERSION)

	srvConf := srv.Config()

	cfg := minecraft.ListenConfig{
		StatusProvider: minecraft.NewStatusProvider("Aether", srvConf.MOTD),
	}

	listener, err := cfg.Listen("raknet", ":19132")
	if err != nil {
		log.Errorlnf("Aether Bedrock Support: error listening: %v", err)
		return
	}
	log.Infolnf("Aether Bedrock Support: listening on %s", listener.Addr())

	p.srv = srv

	go func() {
		if p.closed {
			return
		}
		c, err := listener.Accept()
		if err != nil {
			log.Errorlnf("Aether Bedrock Support: error listening: %v", err)
			return
		}
		conn := c.(*minecraft.Conn)
		go session.HandleNewConn(p.srv, conn)
	}()
}

func (p *Plugin) Unload() {
	p.closed = true
}

func (*Plugin) Identifier() string {
	return "AetherPluginSupport"
}

var _ server.Plugin = (*Plugin)(nil)

var AetherPluginExport = Plugin{}
