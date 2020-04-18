// logstf does mostly ETL stuff for the logstf data
package logstf

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/leighmacdonald/steamid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type HealingSummary struct {
	Healing              int64
	Charges              map[Medigun]int
	ChargeLengths        []float64
	Drops                int
	AvgTimeToBuild       int
	AvgTimeBeforeUsing   int
	NearFullChargeDeaths int
	DeathsAfterCharge    int
	MajorAdvantagesLost  int
	BiggestAdvantageLost int
	timesUntilHeal       []time.Duration
	Targets              map[*Player]int64
	lastEmptyUber        time.Time
}

func (h *HealingSummary) Table(name string) string {
	var dt [][]string
	dt = append(dt, []string{"Healing", fmt.Sprintf("%d", h.Healing)})
	var gs []string
	for k, v := range h.Charges {
		gs = append(gs, fmt.Sprintf("%s: %d", medigunStr(k)[0:1], v))
	}
	dt = append(dt, []string{"Charges", strings.Join(gs, ", ")})
	dt = append(dt, []string{"Drops", fmt.Sprintf("%d", h.Drops)})
	dt = append(dt, []string{"Avg. Build Time", fmt.Sprintf("%d", 0)})
	dt = append(dt, []string{"Avg. Time To Use", fmt.Sprintf("%d", 0)})
	dt = append(dt, []string{"Near Full Deaths", fmt.Sprintf("%d", h.NearFullChargeDeaths)})
	dt = append(dt, []string{"Avg Uber Len.", fmt.Sprintf("%d", 0)})
	dt = append(dt, []string{"Deaths After Charge", fmt.Sprintf("%d", 0)})
	dt = append(dt, []string{"Maj. Adv. Lost", fmt.Sprintf("%d", h.MajorAdvantagesLost)})
	dt = append(dt, []string{"Biggest Adv. Lost", fmt.Sprintf("%d", 0)})
	dt = append(dt, []string{"Heal Targets", ""})

	for _, p := range sortPlayersByHealing(h.Targets) {
		pct := (float64(h.Targets[p]) / float64(h.Healing)) * 100
		dt = append(dt, []string{
			fmt.Sprintf("%s", p.Name),
			fmt.Sprintf("%d (%.0f%%)", h.Targets[p], pct),
		})
	}

	opts := DefaultTableOpts()
	opts.Title = fmt.Sprintf("Healing %s", name)
	return ToTable(dt, opts)
}

type kv struct {
	Key   *Player
	Value int64
}

func sortPlayersByHealing(targets map[*Player]int64) []*Player {
	var ss []kv
	for k, v := range targets {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	var sP []*Player
	for _, p := range ss {
		sP = append(sP, p.Key)
	}
	return sP
}

func (h *HealingSummary) AvgUberLen() float64 {
	if len(h.ChargeLengths) == 0 {
		return 0
	}
	var cl float64
	for _, l := range h.ChargeLengths {
		cl += l
	}
	return cl / float64(len(h.ChargeLengths))
}

func NewHealingSummary() *HealingSummary {
	return &HealingSummary{Targets: map[*Player]int64{}, Charges: map[Medigun]int{}}
}

func medigunStr(medigun Medigun) string {
	switch medigun {
	case kritzkrieg:
		return "Kritzkrieg"
	case vaccinator:
		return "Vaccinator"
	case quickFix:
		return "Quick-Fix"
	default:
		return "Uber"
	}
}

func (s *LogSummary) PrintHealing() {
	var tables [][]string
	for _, med := range s.GetPlayersByClass(medic) {
		tables = append(tables, strings.Split(med.HealingSum.Table(med.Name), "\n"))
	}
	if tables == nil {
		return
	}
	var rows []string
	for i := 0; i <= len(tables[0]); i++ {
		var cols []string
		for _, t := range tables {
			if i < len(t) {
				cols = append(cols, t[i])
			} else {
				cols = append(cols, "")
			}
		}
		rows = append(rows, strings.Join(cols, "   "))
	}
	fmt.Println(strings.Join(rows, "\n"))
}

// Kill tracks the number, via instances, and positions of the attacker/victim
type Kill struct {
	APOS      Position
	VPOS      Position
	Victim    steamid.SID64
	CreatedOn time.Time
}

// Player represents a player on the server. The base properties are global across the
// match.
type Player struct {
	Name           string
	SteamId        steamid.SID64
	Team           Team
	Kills          []Kill
	Deaths         []Kill
	Assists        int
	Revenges       int
	Dominations    int
	Dominated      int
	Healed         int64 // self healing, not medic healing
	Damage         int64
	DamageTaken    int64
	SmallMedPacks  int
	MediumMedPacks int
	FullMedPacks   int
	ShotsFired     int
	ShotsHit       int
	BackStabs      int
	HeadShots      int
	AirShots       int
	Captures       int
	Defenses       int
	Classes        map[PlayerClass]classStats // Classes we have played
	HealingSum     *HealingSummary            // Medic players will get a healing summary
	CurrentClass   PlayerClass
	summary        *LogSummary // Keep reference to get the match times for per min calc
}

type classStats struct {
	Kills     int
	Assist    int
	Deaths    int
	Damage    int
	TotalTime time.Duration
}

func NewPlayer(sum *LogSummary) *Player {
	return &Player{summary: sum, Classes: make(map[PlayerClass]classStats)}
}

func (p *Player) AddClass(cls PlayerClass) {
	_, found := p.Classes[cls]
	if !found {
		p.Classes[cls] = classStats{}
		if cls == medic {
			p.HealingSum = NewHealingSummary()
		}
	}
	p.CurrentClass = cls
}

func (p *Player) DamagePerMin() float64 {
	return float64(p.Damage) / p.summary.TotalLength().Minutes()
}

func (p *Player) DamageTakenPerMin() float64 {
	return float64(p.DamageTaken) / p.summary.TotalLength().Minutes()
}

func (p *Player) Packs() int {
	return p.SmallMedPacks + p.MediumMedPacks + p.FullMedPacks
}

// KAD returns kills and assists per death
func (p *Player) KAD() float64 {
	return float64(len(p.Kills)+p.Assists) / float64(len(p.Deaths))
}

// KD returns kills per death
func (p *Player) KD() float64 {
	return float64(len(p.Kills)) / float64(len(p.Deaths))
}

type RoundSummary struct {
	Length    time.Duration
	LengthRt  time.Duration
	ScoreRed  int
	ScoreBlu  int
	KillsRed  int
	KillsBlu  int
	UbersRed  int
	UbersBlu  int
	DamageRed int64
	DamageBlu int64
	Winner    Team
	MidFight  Team // SPEC == nobody has capped mid yet for the round
}

type TeamSummary struct {
	Kills     int
	Damage    int64
	Charges   int
	Drops     int
	Caps      int
	MidFights int
}

type Message struct {
	Player    *Player
	TeamChat  bool
	Message   string
	Timestamp time.Time
}

type LogSummary struct {
	Id                  int
	Players             map[steamid.SID64]*Player
	Teams               map[Team]*TeamSummary
	MatchName           string
	ServerName          string
	Map                 string
	ScoreRed            int
	ScoreBlu            int
	Duration            time.Duration
	CreatedOn           time.Time
	Rounds              []*RoundSummary
	Messages            []Message
	roundStarted        bool
	roundStartTime      time.Time
	currentRound        int
	currentRoundSummary *RoundSummary
	lastPause           time.Time
	lastPauseDuration   time.Duration
	paused              bool
}

func (s *LogSummary) isRoundStarted() bool {
	return s.roundStarted
}

func NewSummary() *LogSummary {
	return &LogSummary{
		Players: make(map[steamid.SID64]*Player),
		Teams: map[Team]*TeamSummary{
			RED: {},
			BLU: {},
		},
		roundStarted: false,
		currentRound: 1,
	}
}

func (s *LogSummary) getTeamSummary(team Team) *TeamSummary {
	t, found := s.Teams[team]
	if !found {
		t = &TeamSummary{}
		s.Teams[team] = t
	}
	return t
}

func (s *LogSummary) getPlayer(steamId steamid.SID64) *Player {
	if !steamId.Valid() {
		return nil
	}
	player, found := s.Players[steamId]
	if !found {
		player = NewPlayer(s)
		s.Players[steamId] = player
	}
	return player
}

func (s *LogSummary) PrintPlayers(sortBy SortAttr) {
	var dt [][]string
	opts := DefaultTableOpts()
	opts.Title = fmt.Sprintf("Logs for match #%d [len: %s] RED: %d BLU: %d", s.Id, s.TotalLength(), s.ScoreRed,
		s.ScoreBlu)
	headers := []string{
		"Team", "Name", "Class", "K", "A", "D", "DA", "DA/M", "KA/D", "K/D", "DT", "DT/M", "HP", "BS", "HS", "AS",
		"CAP",
	}
	dt = append(dt, headers)
	var players []*Player
	for _, p := range s.Players {
		players = append(players, p)
	}
	sortPlayers(players, sortHP)
	for _, p := range players {
		var pd []string
		pd = append(pd, getTeamStr(p.Team))
		pd = append(pd, p.Name)
		var classes []string
		for c := range p.Classes {
			classes = append(classes, playerClassStr(c)[0:2])
		}
		pd = append(pd, strings.Join(classes, ", "))
		pd = append(pd, fmt.Sprintf("%d", len(p.Kills)))
		pd = append(pd, fmt.Sprintf("%d", p.Assists))
		pd = append(pd, fmt.Sprintf("%d", len(p.Deaths)))
		pd = append(pd, fmt.Sprintf("%d", p.Damage))
		pd = append(pd, fmt.Sprintf("%.1f", p.DamagePerMin()))
		pd = append(pd, fmt.Sprintf("%.1f", p.KAD()))
		pd = append(pd, fmt.Sprintf("%.1f", p.KD()))
		pd = append(pd, fmt.Sprintf("%d", p.DamageTaken))
		pd = append(pd, fmt.Sprintf("%.1f", p.DamageTakenPerMin()))
		pd = append(pd, fmt.Sprintf("%d", p.Packs()))
		pd = append(pd, fmt.Sprintf("%d", p.BackStabs))
		pd = append(pd, fmt.Sprintf("%d", p.HeadShots))
		pd = append(pd, fmt.Sprintf("%d", p.AirShots))
		pd = append(pd, fmt.Sprintf("%d", p.Captures))
		dt = append(dt, pd)
	}
	fmt.Printf(ToTable(dt, opts))
}

type SortAttr int

const (
	sortTeam SortAttr = iota
	sortName
	sortClasses
	sortKills
	sortAssists
	sortDeaths
	sortDamage
	sortDamageMin
	sortKAD
	sortKD
	sortDT
	sortDTM
	sortHP
	sortBS
	sortHS
	sortAS
	sortCAP
)

func sortPlayers(players []*Player, attr SortAttr) {
	sort.SliceStable(players, func(i, j int) bool {
		switch attr {
		case sortTeam:
			return players[i].Team > players[j].Team
		case sortName:
			return players[i].Name > players[j].Name
		case sortClasses:
			panic("not implemented")
			//return players[i].Classes > players[j].Classes
		case sortKills:
			return len(players[i].Kills) > len(players[j].Kills)
		case sortDeaths:
			return len(players[i].Deaths) > len(players[j].Deaths)
		case sortAssists:
			return players[i].Assists > players[j].Assists
		case sortDamage:
			return players[i].Damage > players[j].Damage
		case sortDamageMin:
			return players[i].DamagePerMin() > players[j].DamagePerMin()
		case sortKD:
			return players[i].KD() > players[j].KD()
		case sortKAD:
			return players[i].KAD() > players[j].KAD()
		case sortDT:
			return players[i].DamageTaken > players[j].DamageTaken
		case sortHP:
			return players[i].Packs() > players[j].Packs()
		case sortBS:
			return players[i].BackStabs > players[j].BackStabs
		case sortHS:
			return players[i].HeadShots > players[j].HeadShots
		case sortAS:
			return players[i].AirShots > players[j].AirShots
		case sortCAP:
			return players[i].Captures > players[j].Captures
		case sortDTM:
			return players[i].DamageTakenPerMin() > players[j].DamageTakenPerMin()
		default:
			return players[i].Team > players[j].Team
		}
	})
}

func (s *LogSummary) GetPlayersByClass(class PlayerClass) []*Player {
	var p []*Player
	for _, player := range s.Players {
		for c := range player.Classes {
			if c == class {
				p = append(p, player)
				break
			}
		}
	}
	return p
}

// Apply will parse the input line and send the results to the appropriate method to apply the
// state update.1
//
// NOTE We are relying on the rx engine to match group names, so we dont check most keys first
func (s *LogSummary) Apply(line string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	var (
		player1 *Player
		player2 *Player
	)
	d, msgType := parseLine(line)
	if msgType == skipped {
		return
	}
	if msgType == unhandledMsg {
		log.Warnf("Unhandled message: %s", line)
		return
	}
	p1SId, ok := d["sid"]
	if ok {
		player1 = s.getPlayer(steamid.SID3ToSID64(steamid.SID3(p1SId)))
		if player1 != nil && player1.Name == "" {
			player1.Name = d["name"]
		}
	}
	p2SId, ok := d["sid2"]
	if ok {
		player2 = s.getPlayer(steamid.SID3ToSID64(steamid.SID3(p2SId)))
		if player2 != nil && player2.Name == "" {
			player2.Name = d["name2"]
		}
	}
	dt := parseDateTime(d["date"], d["time"])
	switch msgType {
	case connected:
	case validated:
	case entered:
	case joinedTeam:
		s.joinTeam(player1, parseTeam(d["team"]))
	case changeClass:
		// Spawned as seems to be what we actually want
		s.spawnedAs(player1, parsePlayerClass(d["class"]))
	case spawnedAs:
		s.spawnedAs(player1, parsePlayerClass(d["class"]))
		s.joinTeam(player1, parseTeam(d["team"]))
	case suicide:
		s.suicide(player1, parsePos(d["pos"]), dt)
	case shotFired:
		s.shotFired(player1, d["weapon"])
	case shotHit:
		s.shotHit(player1, d["weapon"])
	case damage:
		weapon := ""
		healing := int64(0)
		realDamage := int64(0)
		damage := int64(0)
		airshot := false
		params := parseParams(d["body"])
		for i, p := range params {
			var err error
			switch p {
			case "damage":
				if realDamage > 0 {
					continue
				}
				damage, err = strconv.ParseInt(params[i+1], 10, 64)
				if err != nil {
					log.Warnf("Failed to parse damage: %s", params[i+1])
					return
				}
			case "realdamage":
				// Real damage is counted for back stabs
				realDamage, err = strconv.ParseInt(params[i+1], 10, 64)
				if err != nil {
					log.Warnf("Failed to parse realdamage: %s", params[i+1])
					return
				}
			case "weapon":
				weapon = params[i+1]
			case "healing":
				healing, err = strconv.ParseInt(params[i+1], 10, 64)
				if err != nil {
					log.Warnf("Failed to parse healing: %s", params[i+1])
					return
				}
			case "airshot":
				airshot = true
			}
		}
		if player1 == nil {
			break
		}
		if isRealDamageWeapon(weapon) && realDamage > 0 {
			s.damage(player1, realDamage, weapon, player2)
		} else {
			s.damage(player1, damage, weapon, player2)
		}
		// Some attacks will heal as well
		if healing > 0 {
			s.selfHealed(player1, healing)
		}
		if airshot {
			s.airShot(player1)
		}
	case killedCustom:
		// TODO just move this to damage and parseAttr?
		if d["customkill"] == "headshot" {
			s.headShot(player1, parsePos(d["apos"]), d["weapon"], player2, parsePos(d["vpos"]), dt)
		} else if d["customkill"] == "backstab" {
			s.backStab(player1, parsePos(d["apos"]), d["weapon"], player2, parsePos(d["vpos"]), dt)
		}
	case killed:
		s.killed(player1, parsePos(d["apos"]), d["weapon"], player2, parsePos(d["vpos"]), dt)
	case killAssist:
		s.assist(player1, parsePos(d["aspos"]), player2, parsePos(d["apos"]))
	case domination:
		s.domination(player1, player2)
	case revenge:
		s.revenge(player1)
	case pickup:
		if player1 == nil {
			break
		}
		if strings.Contains(d["item"], "ammo") {
			_ = parseAmmoPack(d["item"])
		} else {
			hp := parseHealthPack(d["item"])
			switch hp {
			case hpSmall:
				player1.SmallMedPacks++
			case hpMedium:
				player1.MediumMedPacks += 2
			case hpLarge:
				player1.FullMedPacks += 4
			}
		}
	case say:
		s.say(player1, dt, d["msg"], false)
	case sayTeam:
		s.say(player1, dt, d["msg"], true)
	case emptyUber:
		s.emptyUber(player1, dt)
	case medicDeath:
		if d["uber"] == "1" {
			s.chargeDropped(player2)
		}
	case medicDeathEx:
		pct, err := strconv.ParseInt(d["pct"], 10, 64)
		if err != nil {
			log.Warnf("Failed to parse duration:%s", d["duration"])
			break
		}
		if pct > 80 {
			s.chargeAlmostDropped(player1)
		}
	case lostUberAdv:
		s.lostAdvantage(player1)
	case chargeReady:
	case chargeDeployed:
		s.chargeDeployed(player1, parseMedigun(d["medigun"]))
	case chargeEnded:
		duration, err := strconv.ParseFloat(d["duration"], 64)
		if err != nil {
			log.Warnf("Failed to parse duration:%s", d["duration"])
			break
		}
		s.chargeEnded(player1, duration)
	case healed:
		if player1 == nil {
			return
		}
		healing, err := strconv.ParseInt(d["healing"], 10, 64)
		if err != nil {
			log.Warnf("Failed to parse healing:%s", line)
			break
		}
		if player1.CurrentClass == medic {
			s.healed(player1, player2, healing)
		}
		// TODO record sandvich/other healing items
	case extinguished:
	case builtObject:
	case carryObject:
	case killedObject:
	case detonatedObject:
	case dropObject:
	case firstHealAfterSpawn:
		ht, err := strconv.ParseFloat(d["healtime"], 64)
		if err != nil {
			return
		}
		d, err := time.ParseDuration(fmt.Sprintf("%.2fs", ht))
		if err != nil {
			return
		}
		s.firstHealTime(player1, d)
	case pointCaptured:
		params := parseParams(d["body"])
		var players []*Player
		for i, p := range params {
			if strings.HasPrefix(p, "player") {
				m := rxPlayer.FindStringSubmatch(params[i+1])
				if len(m) < 4 {
					log.Warnf("Failed to parse SID from: %s", params[i+1])
					return
				}
				players = append(players, s.getPlayer(steamid.SID3ToSID64(steamid.SID3(m[3]))))
			} else if strings.HasPrefix(p, "position") {
				// Useful at all?
			}
		}
		pc, err := strconv.ParseInt(d["numcappers"], 10, 64)
		if err != nil {
			log.Warnf("Failed to parse numcappers: %s", line)
			break
		}
		if len(players) == 0 {
			break
		}
		if int64(len(players)) != pc {
			log.Warnf("Didnt parse matching player count: %s", line)
			break
		}
		for _, p := range players {
			s.pointCapture(p)
		}
		if s.currentRoundSummary.MidFight == SPEC {
			s.currentRoundSummary.MidFight = players[0].Team
		}
	case captureBlocked:
		s.captureBlocked(player1)
	case wRoundWin:
		winner := parseTeam(d["winner"])
		s.wRoundWin(dt, winner)
	case wRoundLen:
		dur, err := time.ParseDuration(fmt.Sprintf("%ss", d["len"]))
		if err != nil {
			log.Warnf("Failed to parse round len")
			break
		}
		s.wRoundLen(dur, dt.Sub(s.roundStartTime))
	case wRoundStart:
		s.wRoundStart(dt)
	case wPaused:
		s.pause(dt)
	case wUnpaused:
		s.unpause(dt)
	}

}

// parseLine iterates over the regex parsers and if a match is found will return a
// map with the named groups as keys and the MsgType that was matched.
func parseLine(logLine string) (map[string]string, MsgType) {
	for _, p := range rxParsers {
		m, found := reSubMatchMap(p.Rx, logLine)
		if found {
			return m, p.Type
		}
	}
	return nil, unhandledMsg
}

func (s *LogSummary) LoadApiResponse(r *ApiResponse) error {
	s.MatchName = r.Info.Title
	s.Map = r.Info.Map
	return nil
}

// readLog handles reading and transforming the match from a file on disk into a populated LogSummary instance.
// it will accept both a zip file and plain text log file as inputs.
func readLog(logId int64) (*LogSummary, error) {
	rawLogPath := path.Join(cacheDir, LogCacheFile(logId, ZipFormat))
	ls := NewSummary()
	if strings.HasSuffix(strings.ToLower(rawLogPath), "zip") {
		zf, err := zip.OpenReader(rawLogPath)
		if err != nil {
			return ls, err
		}
		if len(zf.File) == 0 {
			return ls, errors.New("no files found in zip archive")
		}
		ff, err := zf.File[0].Open()
		if err != nil {
			return ls, err
		}
		content, err := ioutil.ReadAll(ff)
		if err != nil {
			return ls, err
		}
		for _, line := range strings.Split(string(content), "\n") {
			if line != "" && line != "\r" {
				ls.Apply(line)
			}
		}
	} else {
		file, err := os.Open(rawLogPath)
		if err != nil {
			return ls, err
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.WithError(err).Errorf("Failed to close logstf fh")
			}
		}()
		scanner := bufio.NewScanner(file)
		var v string
		for scanner.Scan() {
			v = scanner.Text()
			ls.Apply(v)
		}
	}
	return ls, nil
}

func ReadJSON(logId int64) (*ApiResponse, error) {
	var ar *ApiResponse
	rawLogPath := path.Join(cacheDir, LogCacheFile(logId, JSONFormat))
	b, err := ioutil.ReadFile(rawLogPath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, ar); err != nil {
		return nil, err
	}
	return ar, nil
}

func Get(logId int64) (*LogSummary, error) {
	sum, err := readLog(logId)
	if err != nil {
		return nil, err
	}
	ar, err := ReadJSON(logId)
	if err != nil {
		return nil, err
	}
	if err := sum.LoadApiResponse(ar); err != nil {
		log.Warnf("Failed to read api response")
		// Stop if we dont have api resp?
		//return sum, nil
	}
	return sum, nil
}
