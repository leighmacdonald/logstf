package logstf

import (
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// Rollback will rollback a transaction, logging any errors that happen
func Rollback(tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil {
		log.WithError(err).Errorln("Failed to rollback transaction")
	}
}

func insertMatch(tx *sqlx.Tx, s *LogSummary) error {
	qMatch := `INSERT INTO logstf_match (log_id, score_red, score_blu, title, duration, map, created_on) 
			    VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.Exec(qMatch, s.Id, s.ScoreRed, s.ScoreBlu, s.MatchName, s.Map, s.CreatedOn)
	if err != nil {
		return err
	}
	return nil
}

func insertPlayers(tx *sqlx.Tx, s *LogSummary) error {
	for sid, player := range s.Players {
		stmt, err := tx.Preparex(`INSERT INTO logstf_player (log_id, steam_id, team, kills, assists, revenges, dominations, 
                           dominated, healed, damage, damage_taken, small_med_packs, medium_med_packs, large_med_packs, 
                           shots_fired, shots_hit, backstabs, headshots, airshots, captures, defenses) VALUES ($1, 
$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`)
		if err != nil {
			Rollback(tx)
			return err
		}
		_, err = stmt.Exec(s.Id, sid.Int64(), player.Team, player.Kills, player.Assists, player.Revenges,
			player.Dominations, player.Dominated, player.Healed, player.Damage, player.DamageTaken,
			player.SmallMedPacks, player.MediumMedPacks, player.FullMedPacks, player.ShotsFired, player.ShotsHit,
			player.BackStabs, player.HeadShots, player.AirShots, player.Captures, player.Defenses)
		if err != nil {
			Rollback(tx)
			return err
		}
	}
	return nil
}

func insertPlayerClasses(tx *sqlx.Tx, s *LogSummary) error {
	for sid, player := range s.Players {
		for _, cls := range player.Classes {
			stmt, err := tx.Preparex(`
			INSERT INTO logstf_player_classes (
				log_id, steam_id, class_id
			) VALUES ($1, $2, $3)`)
			if err != nil {
				Rollback(tx)
				return err
			}
			_, err = stmt.Exec(s.Id, sid.Int64(), cls)
			if err != nil {
				Rollback(tx)
				return err
			}
		}
	}
	return nil
}

func insertPlayerKills(tx *sqlx.Tx, s *LogSummary) error {
	for sid, player := range s.Players {
		for _, kill := range player.Kills {
			stmt, err := tx.Preparex(`
			INSERT INTO logstf_player_kills (
				log_id, steam_id, victim_steam_id, attacker_pos, victim_pos, created_on
			) VALUES ($1, $2, $3, ST_MakePoint($4, $5, $6), ST_MakePoint($7, $8, $9), $10)`)
			if err != nil {
				Rollback(tx)
				return err
			}
			_, err = stmt.Exec(s.Id, sid.Int64(), kill.Victim.Int64(), kill.APOS.X, kill.APOS.Y, kill.APOS.Z,
				kill.VPOS.X, kill.VPOS.Y, kill.VPOS.Z, kill.CreatedOn)
			if err != nil {
				Rollback(tx)
				return err
			}
		}
	}
	return nil
}

func Insert(conn *sqlx.DB, s *LogSummary) error {
	tx, err := conn.Beginx()
	if err != nil {
		return err
	}
	if err := insertMatch(tx, s); err != nil {
		Rollback(tx)
		return err
	}
	if err := insertPlayers(tx, s); err != nil {
		Rollback(tx)
		return err
	}
	if err := insertPlayerClasses(tx, s); err != nil {
		Rollback(tx)
		return err
	}
	if err := insertPlayerKills(tx, s); err != nil {
		Rollback(tx)
		return err
	}
	if err := tx.Commit(); err != nil {
		Rollback(tx)
		return err
	}
	return nil
}
