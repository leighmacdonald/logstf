package logstf

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MsgType int

const (
	connected MsgType = iota
	disconnected
	validated
	entered
	joinedTeam
	changeClass
	spawnedAs
	suicide
	shotFired
	shotHit
	damage
	killed
	killAssist
	domination
	revenge
	pickup
	say
	sayTeam
	emptyUber
	medicDeath
	medicDeathEx
	lostUberAdv
	unhandledMsg
	chargeReady
	chargeDeployed
	chargeEnded
	healed
	extinguished
	builtObject
	carryObject
	killedObject
	detonatedObject
	dropObject
	firstHealAfterSpawn
	captureBlocked
	killedCustom
	pointCaptured
	wRoundOvertime
	// World events not attached to specific players
	wRoundStart
	wRoundWin
	wRoundLen
	wTeamScore
	wTeamFinalScore
	wGameOver
	wPaused
	wUnpaused
	// Any messages we skip processing
	skipped
)

type HealthPack int

const (
	hpSmall HealthPack = iota
	hpMedium
	hpLarge
)

func parseHealthPack(hp string) HealthPack {
	switch hp {
	case "medkit_small":
		return hpSmall
	case "medkit_medium":
		return hpMedium
	case "medkit_full":
		return hpLarge
	default:
		return hpMedium
	}
}

type AmmoPack int

const (
	ammoSmall AmmoPack = iota
	ammoMedium
	ammoLarge
)

func parseAmmoPack(hp string) AmmoPack {
	switch hp {
	case "tf_ammo_pack":
		return ammoSmall
	case "ammopack_medium":
		return ammoMedium
	default:
		return ammoLarge
	}
}

type PlayerClass int

const (
	spectator PlayerClass = iota
	scout
	soldier
	pyro
	demo
	heavy
	engineer
	medic
	sniper
	spy
)

type Medigun int

const (
	uber Medigun = iota
	kritzkrieg
	vaccinator
	quickFix
)

func parseMedigun(gunStr string) Medigun {
	switch strings.ToLower(gunStr) {
	case "medigun":
		return uber
	case "kritzkrieg":
		return kritzkrieg
	case "vaccinator":
		return vaccinator
	default:
		return quickFix
	}
}

func playerClassStr(cls PlayerClass) string {
	switch cls {
	case scout:
		return "Scout"
	case soldier:
		return "Soldier"
	case pyro:
		return "Pyro"
	case heavy:
		return "Heavy"
	case engineer:
		return "Engineer"
	case medic:
		return "Medic"
	case sniper:
		return "Sniper"
	case spy:
		return "Spy"
	default:
		return "Spectator"
	}
}

func parsePlayerClass(classStr string) PlayerClass {
	switch strings.ToLower(classStr) {
	case "scout":
		return scout
	case "soldier":
		return soldier
	case "pyro":
		return pyro
	case "demoman":
		return demo
	case "heavyweapons":
		return heavy
	case "engineer":
		return engineer
	case "medic":
		return medic
	case "sniper":
		return sniper
	case "spy":
		return spy
	default:
		return spectator
	}
}

type Team uint8

const (
	SPEC Team = 0
	RED  Team = 1
	BLU  Team = 2
)

func getTeamStr(t Team) string {
	switch t {
	case RED:
		return "RED"
	case BLU:
		return "BLU"
	default:
		return "SPEC"
	}
}

var (
	rxPlayer  *regexp.Regexp
	rxParsers []parserType
)

type parserType struct {
	Rx   *regexp.Regexp
	Type MsgType
}

func reSubMatchMap(r *regexp.Regexp, str string) (map[string]string, bool) {
	match := r.FindStringSubmatch(str)
	subMatchMap := make(map[string]string)
	if match == nil {
		return nil, false
	}
	for i, name := range r.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}
	return subMatchMap, true
}

func parseTeam(team string) Team {
	t := SPEC
	if team == "Red" {
		t = RED
	} else if team == "Blue" {
		t = BLU
	} else {
		t = SPEC
	}
	return t
}

type Position struct {
	X int64
	Y int64
	Z int64
}

func parsePos(pos string) Position {
	p := strings.SplitN(pos, " ", 3)
	x, err := strconv.ParseInt(p[0], 10, 64)
	if err != nil {
		log.Warnf("Failed to parse x pos: %s", p[0])
		x = 0
	}
	y, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil {
		log.Warnf("Failed to parse y pos: %s", p[1])
		y = 0
	}
	z, err := strconv.ParseInt(p[2], 10, 64)
	if err != nil {
		log.Warnf("Failed to parse z pos: %s", p[2])
		z = 0
	}
	return Position{x, y, z}
}

func parseDateTime(dateStr, timeStr string) time.Time {
	fDateStr := fmt.Sprintf("%s %s", dateStr, timeStr)
	t, err := time.Parse("02/01/2006 15:04:05", fDateStr)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse date: %s", fDateStr)
		return time.Now()
	}
	return t
}

func parseParams(body string) []string {
	b := strings.ReplaceAll(strings.ReplaceAll(body, "(", ""), "\"", "")
	pcs := strings.Split(strings.ReplaceAll(b, ")", ""), " ")
	return pcs
}

func isRealDamageWeapon(weapon string) bool {
	weapons := []string{"big_earner", "black_rose", "eternal_reward", "knife", "kunai", "sharp_dresser", "spy_cicle"}
	for _, realWeapon := range weapons {
		if weapon == realWeapon {
			return true
		}
	}
	return false
}

func init() {
	// Date stuff
	rxDate := `^L\s(?P<date>.+?)\s+-\s+(?P<time>.+?):\s+`
	// Common player id format eg: "funk. Bubi<382><STEAM_0:1:22649331><>"
	rxPlayerStr := `"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator))?>"`
	rxPlayer = regexp.MustCompile(`(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator)?)>`)
	// Most player events have the same common prefix
	dp := rxDate + rxPlayerStr + `\s+`

	rxSkipped := regexp.MustCompile(`("undefined"$)`)
	rxConnected := regexp.MustCompile(dp + `connected, address`)
	rxDisconnected := regexp.MustCompile(dp + `disconnected \(reason "(?P<reason>.+?)"\)`)
	rxValidated := regexp.MustCompile(dp + `STEAM USERID validated$`)
	rxEntered := regexp.MustCompile(dp + `entered the game`)
	rxJoinedTeam := regexp.MustCompile(dp + `joined team "(?P<team>(Red|Blue|Spectator))"`)
	rxChangeClass := regexp.MustCompile(dp + `changed role to "(?P<class>.+?)"`)
	rxSpawned := regexp.MustCompile(dp + `spawned as "(?P<class>\S+)"`)
	rxSuicide := regexp.MustCompile(dp + `committed suicide with "world" \(attacker_position "(?P<pos>.+?)"\)`)
	rxShotFired := regexp.MustCompile(dp + `triggered "shot_fired" \(weapon "(?P<weapon>\S+)"\)`)
	rxShotHit := regexp.MustCompile(dp + `triggered "shot_hit" \(weapon "(?P<weapon>\S+)"\)`)
	rxDamage := regexp.MustCompile(dp + `triggered "damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s(?P<body>.+?)$`)
	//rxDamageRealHeal := regexp.MustCompile(dp + `triggered "damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(damage "(?P<damage>\d+)"\) \(realdamage "(?P<realdamage>\d+)"\) \(weapon "(?P<weapon>.+?)"\) \(healing "(?P<healing>\d+)"\)`)
	// rxDamage := regexp.MustCompile(dp + `triggered "damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>".+?damage "(?P<damage>\d+)"\) \(weapon "(?P<weapon>\S+)"\)`)
	// Old format only?
	rxDamageOld := regexp.MustCompile(dp + `triggered "damage" \(damage "(?P<damage>\d+)"\)`)
	rxKilled := regexp.MustCompile(dp + `killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxKilledCustom := regexp.MustCompile(dp + `killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(customkill "(?P<customkill>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxAssist := regexp.MustCompile(dp + `triggered "kill assist" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(assister_position "(?P<aspos>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxDomination := regexp.MustCompile(dp + `triggered "domination" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>"`)
	rxRevenge := regexp.MustCompile(dp + `triggered "revenge" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s?(\(assist "(?P<assist>\d+)"\))?`)
	rxPickup := regexp.MustCompile(dp + `picked up item "(?P<item>\S+)"`)
	rxSay := regexp.MustCompile(dp + `say\s+"(?P<msg>.+?)"$`)
	rxSayTeam := regexp.MustCompile(dp + `say_team\s+"(?P<msg>.+?)"$`)
	rxEmptyUber := regexp.MustCompile(dp + `triggered "empty_uber"`)
	rxMedicDeath := regexp.MustCompile(dp + `triggered "medic_death" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" \(healing "(?P<healing>\d+)"\) \(ubercharge "(?P<uber>\d+)"\)`)
	rxMedicDeathEx := regexp.MustCompile(dp + `triggered "medic_death_ex" \(uberpct "(?P<pct>\d+)"\)`)
	rxLostUberAdv := regexp.MustCompile(dp + `triggered "lost_uber_advantage" \(time "(?P<time>\d+)"\)`)
	rxChargeReady := regexp.MustCompile(dp + `triggered "chargeready"`)
	rxChargeDeployed := regexp.MustCompile(dp + `triggered "chargedeployed"( \(medigun "(?P<medigun>.+?)"\))?`)
	rxChargeEnded := regexp.MustCompile(dp + `triggered "chargeended" \(duration "(?P<duration>.+?)"\)`)
	rxHealed := regexp.MustCompile(dp + `triggered "healed" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(healing "(?P<healing>\d+)"\)`)
	rxExtinguished := regexp.MustCompile(dp + `triggered "player_extinguished" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>" with "(?P<weapon>.+?)" \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxBuiltObject := regexp.MustCompile(dp + `triggered "player_builtobject" \(object "(?P<object>.+?)"\) \(position "(?P<Position>.+?)"\)`)
	rxCarryObject := regexp.MustCompile(dp + `triggered "player_carryobject" \(object "(?P<object>.+?)"\) \(position "(?P<Position>.+?)"\)`)
	rxDropObject := regexp.MustCompile(dp + `triggered "player_dropobject" \(object "(?P<object>.+?)"\) \(position "(?P<Position>.+?)"\)`)
	rxKilledObject := regexp.MustCompile(dp + `triggered "killedobject" \(object "(?P<object>.+?)"\) \(weapon "(?P<weapon>.+?)"\) \(objectowner "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\) \(attacker_position "(?P<apos>.+?)"\)`)
	rxKilledObjectAssisted := regexp.MustCompile(dp + `triggered "killedobject" \(object "(?P<object>.+?)"\) \(objectowner "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\)\s+\(assist "1"\) \(assister_position "(?P<aspos>.+?)"\) \(attacker_position "(?P<apos>.+?)"\)`)
	rxDetonatedObject := regexp.MustCompile(dp + `triggered "object_detonated" \(object "(?P<object>.+?)"\) \(position "(?P<Position>.+?)"\)`)
	rxFirstHealAfterSpawn := regexp.MustCompile(dp + `triggered "first_heal_after_spawn" \(time "(?P<healtime>.+?)"\)`)
	rxWOvertime := regexp.MustCompile(rxDate + `World triggered "Round_Overtime"`)
	rxWRoundStart := regexp.MustCompile(rxDate + `World triggered "Round_Start"`)
	rxWGameOver := regexp.MustCompile(rxDate + `World triggered "Game_Over" reason "(?P<reason>.+?)"`)
	rxWRoundLen := regexp.MustCompile(rxDate + `World triggered "Round_Length" \(seconds "(?P<len>.+?)"\)`)
	rxWRoundWin := regexp.MustCompile(rxDate + `World triggered "Round_Win" \(winner "(?P<winner>.+?)"\)`)
	rxWTeamFinalScore := regexp.MustCompile(rxDate + `Team "(?P<team>Red|Blue)" final score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxWTeamScore := regexp.MustCompile(rxDate + `Team "(?P<team>Red|Blue)" current score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxCaptureBlocked := regexp.MustCompile(dp + `triggered "captureblocked" \(cp "(?P<cp>\d+)"\) \(cpname "(?P<cpname>.+?)"\) \(position "(?P<pos>.+?)"\)`)
	rxPointCaptured := regexp.MustCompile(rxDate + `Team "(?P<team>.+?)" triggered "pointcaptured" \(cp "(?P<cp>\d+)"\) \(cpname "(?P<cpname>.+?)"\) \(numcappers "(?P<numcappers>\d+)"\)(\s+(?P<body>.+?))$`)
	rxWPaused := regexp.MustCompile(rxDate + `World triggered "Game_Paused"`)
	rxWUnpaused := regexp.MustCompile(rxDate + `World triggered "Game_Unpaused"`)
	// Associate matching rx's with a MsgType
	// Should be ordered by most common events first to reduce running all the rx's as much as possible
	rxParsers = []parserType{
		{rxShotFired, shotFired},
		{rxShotHit, shotHit},
		//{rxDamageRealHeal, damage},
		{rxDamage, damage},
		{rxDamageOld, damage},
		{rxKilled, killed},
		{rxHealed, healed},
		{rxKilledCustom, killedCustom},
		{rxAssist, killAssist},
		{rxPickup, pickup},
		{rxSpawned, spawnedAs},
		{rxValidated, validated},
		{rxConnected, connected},
		{rxEntered, entered},
		{rxJoinedTeam, joinedTeam},
		{rxChangeClass, changeClass},
		{rxSuicide, suicide},
		{rxChargeReady, chargeReady},
		{rxChargeDeployed, chargeDeployed},
		{rxChargeEnded, chargeEnded},
		{rxDomination, domination},
		{rxRevenge, revenge},
		{rxSay, say},
		{rxSayTeam, sayTeam},
		{rxEmptyUber, emptyUber},
		{rxLostUberAdv, lostUberAdv},
		{rxMedicDeath, medicDeath},
		{rxMedicDeathEx, medicDeathEx},
		{rxExtinguished, extinguished},
		{rxBuiltObject, builtObject},
		{rxCarryObject, carryObject},
		{rxDropObject, dropObject},
		{rxKilledObject, killedObject},
		{rxKilledObjectAssisted, killedObject},
		{rxDetonatedObject, detonatedObject},
		{rxFirstHealAfterSpawn, firstHealAfterSpawn},
		{rxPointCaptured, pointCaptured},
		{rxCaptureBlocked, captureBlocked},
		{rxDisconnected, disconnected},
		{rxWOvertime, wRoundOvertime},
		{rxWRoundStart, wRoundStart},
		{rxWRoundWin, wRoundWin},
		{rxWRoundLen, wRoundLen},
		{rxWGameOver, wGameOver},
		{rxWTeamScore, wTeamScore},
		{rxWTeamFinalScore, wTeamFinalScore},
		{rxWPaused, wPaused},
		{rxWUnpaused, wUnpaused},
		{rxSkipped, skipped},
	}
}
