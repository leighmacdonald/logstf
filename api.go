package logstf

import (
	"encoding/json"
	"errors"
	"fmt"
	steam "github.com/leighmacdonald/steamid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type TeamStats struct {
	Score     int `json:"score"`
	Kills     int `json:"kills"`
	Deaths    int `json:"deaths"`
	Dmg       int `json:"dmg"`
	Charges   int `json:"charges"`
	Drops     int `json:"drops"`
	Firstcaps int `json:"firstcaps"`
	Caps      int `json:"caps"`
}

type weaponStats struct {
	Kills  int     `json:"kills"`
	Dmg    int     `json:"dmg"`
	AvgDmg float64 `json:"avg_dmg"`
	Shots  int     `json:"shots"`
	Hits   int     `json:"hits"`
}

type classKills struct {
	Pyro         int `json:"pyro"`
	Heavyweapons int `json:"heavyweapons"`
	Sniper       int `json:"sniper"`
	Demoman      int `json:"demoman"`
	Spy          int `json:"spy"`
	Engineer     int `json:"engineer"`
	Scout        int `json:"scout"`
	Medic        int `json:"medic"`
	Soldier      int `json:"soldier"`
}

type uberTypes struct {
	Kritzkrieg int `json:"kritzkrieg"`
	Medigun    int `json:"medigun"`
	QuickFix   int `json:"quickfix"`
	Vaccinator int `json:"vaccinator"`
}

type playerStats struct {
	Team       string `json:"team"`
	ClassStats []struct {
		Type      string                 `json:"type"`
		Kills     int                    `json:"kills"`
		Assists   int                    `json:"assists"`
		Deaths    int                    `json:"deaths"`
		Dmg       int                    `json:"dmg"`
		Weapon    map[string]weaponStats `json:"weapon"`
		TotalTime int                    `json:"total_time"`
	} `json:"class_stats"`
	Kills        int       `json:"kills"`
	Deaths       int       `json:"deaths"`
	Assists      int       `json:"assists"`
	Suicides     int       `json:"suicides"`
	Kapd         string    `json:"kapd"`
	Kpd          string    `json:"kpd"`
	Dmg          int64     `json:"dmg"`
	DmgReal      int64     `json:"dmg_real"`
	Dt           int64     `json:"dt"`
	DtReal       int64     `json:"dt_real"`
	Hr           int       `json:"hr"`
	Lks          int       `json:"lks"`
	As           int       `json:"as"`
	Dapd         int       `json:"dapd"`
	Dapm         int       `json:"dapm"`
	Ubers        int       `json:"ubers"`
	Ubertypes    uberTypes `json:"ubertypes"`
	Drops        int       `json:"drops"`
	Medkits      int       `json:"medkits"`
	MedkitsHp    int       `json:"medkits_hp"`
	Backstabs    int       `json:"backstabs"`
	Headshots    int       `json:"headshots"`
	HeadshotsHit int       `json:"headshots_hit"`
	Sentries     int       `json:"sentries"`
	Heal         int       `json:"heal"`
	Cpc          int       `json:"cpc"`
	Ic           int       `json:"ic"`
}

type teamRound struct {
	Score int   `json:"score"`
	Kills int   `json:"kills"`
	Dmg   int64 `json:"dmg"`
	Ubers int   `json:"ubers"`
}

type ApiResponse struct {
	Version int `json:"version"`
	Teams   struct {
		Red  TeamStats `json:"Red"`
		Blue TeamStats `json:"Blue"`
	} `json:"teams"`
	Length  int                    `json:"length"`
	Players map[string]playerStats `json:"players"`
	Names   map[string]string      `json:"names"`
	Rounds  []struct {
		StartTime int    `json:"start_time"`
		Winner    string `json:"winner"`
		Team      struct {
			Blu teamRound `json:"blue"`
			Red teamRound `json:"red"`
		} `json:"team"`
		Events []struct {
			Type    string `json:"type"`
			Time    int    `json:"time"`
			Team    string `json:"team"`
			Steamid string `json:"steamid,omitempty"`
			Killer  string `json:"killer,omitempty"`
			Point   int    `json:"point,omitempty"`
			Medigun string `json:"medigun,omitempty"`
		} `json:"events"`
		//Players interface{} `json:"players"`
		FirstCap string `json:"firstcap"`
		Length   int64  `json:"length"`
	} `json:"rounds"`

	HealSpread map[string]map[string]int `json:"healspread"`

	ClassKills       map[string]classKills `json:"classkills"`
	ClassDeaths      map[string]classKills `json:"classdeaths"`
	ClassKillAssists map[string]classKills `json:"classkillassists"`
	Chat             []struct {
		Steamid string `json:"steamid"`
		Name    string `json:"name"`
		Msg     string `json:"msg"`
	} `json:"chat"`
	Info struct {
		Map             string        `json:"map"`
		Supplemental    bool          `json:"supplemental"`
		TotalLength     int64         `json:"total_length"`
		HasRealDamage   bool          `json:"hasRealDamage"`
		HasWeaponDamage bool          `json:"hasWeaponDamage"`
		HasAccuracy     bool          `json:"hasAccuracy"`
		HasHP           bool          `json:"hasHP"`
		HasHPReal       bool          `json:"hasHP_real"`
		HasHS           bool          `json:"hasHS"`
		HasHSHit        bool          `json:"hasHS_hit"`
		HasBS           bool          `json:"hasBS"`
		HasCP           bool          `json:"hasCP"`
		HasSB           bool          `json:"hasSB"`
		HasDT           bool          `json:"hasDT"`
		HasAS           bool          `json:"hasAS"`
		HasHR           bool          `json:"hasHR"`
		HasIntel        bool          `json:"hasIntel"`
		ADScoring       bool          `json:"AD_scoring"`
		Notifications   []interface{} `json:"notifications"`
		Title           string        `json:"title"`
		Date            int64         `json:"date"`
		Uploader        struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Info string `json:"info"`
		} `json:"uploader"`
	} `json:"info"`
	KillStreaks []killStreak `json:"killstreaks"`
	Success     bool         `json:"success"`
}

type killStreak struct {
	Steamid string `json:"steamid"`
	Streak  int    `json:"streak"`
	Time    int64  `json:"time"`
}

func FetchAPIFile(logId int64, path string) error {
	resp, err := FetchAPI(logId)
	if err != nil {
		return err
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, b, 0755); err != nil {
		return err
	}
	return nil
}

func FetchAPI(logId int64) (*ApiResponse, error) {
	var ar *ApiResponse
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("https://logs.tf/api/v1/log/%d", logId))
	if err != nil {
		return ar, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Failed to close response body ")
		}
	}()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ar, err
	}
	if err := json.Unmarshal(b, &ar); err != nil {
		return ar, err
	}
	if ar.Success == false {
		// When does this occur?
		return ar, errors.New("got non successful response reply")
	}
	return ar, nil
}

func (a *ApiResponse) Summary() *LogSummary {
	s := NewSummary()
	s.Map = a.Info.Map
	s.MatchName = a.Info.Title
	s.CreatedOn = time.Unix(a.Info.TotalLength, 0)
	for sid3, p := range a.Players {
		player := NewPlayer(s)
		player.SteamId = steam.SID3ToSID64(steam.SID3(sid3))
		for i := 0; i < p.Kills; i++ {
			player.Kills = append(player.Kills, Kill{
				APOS:      Position{0, 0, 0},
				VPOS:      Position{0, 0, 0},
				Victim:    0,
				CreatedOn: time.Time{},
			})
		}
		for i := 0; i < p.Deaths; i++ {
			player.Deaths = append(player.Deaths, Kill{
				APOS:      Position{0, 0, 0},
				VPOS:      Position{0, 0, 0},
				Victim:    0,
				CreatedOn: time.Time{},
			})
		}
		for _, cs := range p.ClassStats {
			player.AddClass(parsePlayerClass(cs.Type))
		}
		player.AirShots = p.As
		player.BackStabs = p.Backstabs
		if a.Info.HasRealDamage {
			player.Damage = p.DmgReal
		} else {
			player.Damage = p.Dmg
		}
		player.Captures = p.Cpc
		player.DamageTaken = p.Dt
		s.Players[player.SteamId] = player
	}
	for _, r := range a.Rounds {
		s.Rounds = append(s.Rounds, &RoundSummary{
			Winner:    parseTeam(r.Winner),
			Length:    time.Duration(r.Length) * time.Second,
			LengthRt:  0,
			ScoreRed:  r.Team.Red.Score,
			ScoreBlu:  r.Team.Blu.Score,
			KillsRed:  r.Team.Red.Kills,
			KillsBlu:  r.Team.Blu.Kills,
			UbersRed:  r.Team.Red.Ubers,
			UbersBlu:  r.Team.Blu.Ubers,
			DamageRed: r.Team.Red.Dmg,
			DamageBlu: r.Team.Blu.Dmg,
			MidFight:  parseTeam(r.FirstCap),
		})
	}
	return s
}
