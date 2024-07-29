package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/zeppelinmc/zeppelin/net/packet/play"
)

func handleAnimate(s *BedrockSession, animate *packet.Animate) {
	var animation byte
	switch animate.ActionType {
	case packet.AnimateActionSwingArm:
		animation = play.AnimationSwingMainArm
	case packet.AnimateActionStopSleep:
		animation = play.AnimationLeaveBed
	case packet.AnimateActionCriticalHit:
		animation = play.AnimationCriticalEffect
	case packet.AnimateActionMagicCriticalHit:
		animation = play.AnimationMagicCriticalEffect
	}
	s.srv.Broadcast.Animation(s, animation)
}
