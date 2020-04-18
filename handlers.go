package logstf

import (
	log "github.com/sirupsen/logrus"
	"time"
)

func (s *LogSummary) joinTeam(player *Player, team Team) {
	if team != SPEC {
		player.Team = team
	}
}

func (s *LogSummary) spawnedAs(player *Player, cls PlayerClass) {
	player.AddClass(cls)
}

func (s *LogSummary) headShot(player1 *Player, pos1 Position, weapon string, player2 *Player, pos2 Position, dt time.Time) {
	player1.HeadShots++
	s.killed(player1, pos1, weapon, player2, pos2, dt)
}

func (s *LogSummary) backStab(player1 *Player, pos1 Position, weapon string, player2 *Player, pos2 Position, dt time.Time) {
	player1.BackStabs++
	s.killed(player1, pos1, weapon, player2, pos2, dt)
}

func (s *LogSummary) killed(player1 *Player, pos1 Position, weapon string, player2 *Player, pos2 Position, dt time.Time) {
	if !s.isRoundStarted() {
		return
	}
	player1.Kills = append(player1.Kills, Kill{pos1, pos2, player2.SteamId, dt})
	s.getTeamSummary(player1.Team).Kills++
	if player1.Team == RED {
		s.currentRoundSummary.KillsRed++
	} else if player1.Team == BLU {
		s.currentRoundSummary.KillsBlu++
	}
	player2.Deaths = append(player2.Deaths, Kill{pos1, pos2, player1.SteamId, dt})
}

func (s *LogSummary) suicide(player1 *Player, pos1 Position, dt time.Time) {
	if !s.isRoundStarted() {
		return
	}
	player1.Deaths = append(player1.Deaths, Kill{pos1, pos1, player1.SteamId, dt})
}

func (s *LogSummary) shotFired(player *Player, weapon string) {
	if !s.isRoundStarted() {
		return
	}
	player.ShotsFired++
}

func (s *LogSummary) shotHit(player *Player, weapon string) {
	if !s.isRoundStarted() {
		return
	}
	player.ShotsHit++
}

func (s *LogSummary) assist(player1 *Player, assisterPos Position, player2 *Player, attackerPos Position) {
	player1.Assists++
}

func (s *LogSummary) airShot(player1 *Player) {
	player1.AirShots++
}

func (s *LogSummary) pointCapture(player1 *Player) {
	player1.Captures++
}

func (s *LogSummary) captureBlocked(player1 *Player) {
	player1.Defenses++
}

func (s *LogSummary) damage(player1 *Player, amount int64, weapon string, player2 *Player) {
	if !s.isRoundStarted() {
		return
	}
	// Overall player1 damage
	player1.Damage += amount

	// Overall team damage
	s.getTeamSummary(player1.Team).Damage += amount

	// Current round damage
	switch player1.Team {
	case RED:
		s.currentRoundSummary.DamageRed += amount
	case BLU:
		s.currentRoundSummary.DamageBlu += amount
	}
	if player2 != nil {
		// Not present on older logs?
		player2.DamageTaken += amount
	}
}

// h
func (s *LogSummary) selfHealed(player *Player, amount int64) {
	player.Healed += amount
}

func (s *LogSummary) healed(player1 *Player, player2 *Player, amount int64) {
	if player1.HealingSum == nil {
		log.Warnf("Healing sum nil")
	}
	player1.HealingSum.Healing += amount
	player1.HealingSum.Targets[player2] += amount
}

func (s *LogSummary) wRoundStart(dt time.Time) {
	s.roundStarted = true
	s.roundStartTime = dt
	s.currentRoundSummary = &RoundSummary{
		MidFight: SPEC,
	}
}

func (s *LogSummary) wRoundLen(t time.Duration, trt time.Duration) {
	s.currentRoundSummary.Length = t
	s.currentRoundSummary.LengthRt = trt
	s.currentRound++
}

func (s *LogSummary) wRoundWin(dt time.Time, winner Team) {
	s.roundStarted = false
	if s.currentRoundSummary != nil {
		s.Rounds = append(s.Rounds, s.currentRoundSummary)
	}
	s.currentRoundSummary.LengthRt += dt.Sub(s.roundStartTime)
	if winner == RED {
		s.ScoreRed++
	} else if winner == BLU {
		s.ScoreBlu++
	}
}

func (s *LogSummary) TotalLength() time.Duration {
	var t time.Duration
	for _, r := range s.Rounds {
		t += r.Length
	}
	return t
}

func (s *LogSummary) pickup(player *Player, amount int64) {

}

func (s *LogSummary) revenge(player *Player) {
	player.Revenges++
}

func (s *LogSummary) domination(player1 *Player, player2 *Player) {
	player1.Dominations++
	player2.Dominated++
}

func (s *LogSummary) chargeEnded(player *Player, duration float64) {
	player.HealingSum.ChargeLengths = append(player.HealingSum.ChargeLengths, duration)
}

func (s *LogSummary) chargeDeployed(player *Player, medigun Medigun) {
	player.HealingSum.Charges[medigun]++
}

func (s *LogSummary) chargeDropped(player *Player) {
	if player.HealingSum == nil {
		player.HealingSum = NewHealingSummary()
	}
	player.HealingSum.Drops++
}

func (s *LogSummary) chargeAlmostDropped(player *Player) {
	player.HealingSum.NearFullChargeDeaths++
}

func (s *LogSummary) emptyUber(player *Player, ts time.Time) {
	player.HealingSum.lastEmptyUber = ts
}

func (s *LogSummary) say(player *Player, ts time.Time, message string, teamChat bool) {
	s.Messages = append(s.Messages, Message{
		Player:    player,
		TeamChat:  teamChat,
		Message:   message,
		Timestamp: ts,
	})
}
func (s *LogSummary) pause(ts time.Time) {
	s.lastPause = ts
	s.paused = true
}

func (s *LogSummary) unpause(ts time.Time) {
	// Dont count the duplicate pause/unpause log lines
	if s.paused {
		s.lastPauseDuration = ts.Sub(s.lastPause)
		s.paused = false
	}
}

func (s *LogSummary) firstHealTime(player *Player, d time.Duration) {
	player.HealingSum.timesUntilHeal = append(player.HealingSum.timesUntilHeal, d)
}

func (s *LogSummary) lostAdvantage(player *Player) {
	player.HealingSum.MajorAdvantagesLost++
}
