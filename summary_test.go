package logstf

import (
	"github.com/leighmacdonald/steamid"
	"github.com/stretchr/testify/assert"
	"log"
	"math"
	"path"
	"testing"
)

// getExamplePath looks relative to the current and parent directories for a
// matching file in the example_data folder. It just lets us run tests from project root or
// package roots
func getExamplePath(filename string) string {
	filename1 := path.Join("example_data", filename)
	if !Exists(filename1) {
		filename2 := path.Join("../", "example_data", filename)
		if !Exists(filename2) {
			log.Fatalf("Invalid test data path: %s", filename2)
		}
		filename1 = filename2
	}
	return filename1
}

func TestParsePos(t *testing.T) {
	assert.Equal(t, Position{1, 2, -3}, parsePos("1 2 -3"))
}

func testBaseDir() string {
	if Exists("./example_data") {
		return "./example_data"
	} else if Exists("../example_data") {
		return "../example_data"
	} else {
		return "./"
	}
}

// Should match: http://logs.tf/2325027
func TestReadLog(t *testing.T) {
	// Set the download dir to our example_data folder.
	//config.Get().Set(config.CfgLogsTfCacheDir, testBaseDir())
	log.Panic("Fix for public release")
	// Old style
	ls2, err := readLog(15775)
	assert.NoError(t, err)
	assert.Equal(t, "asdf", ls2.MatchName)

	ls2.PrintPlayers(sortTeam)
	ls2.PrintHealing()
	// Newer style
	ls, err := readLog(2325027)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(ls.Rounds))
	p := ls.Players[steamid.SID64(76561198053921882)]
	assert.Equal(t, 43, len(p.Kills))
	assert.Equal(t, 13, p.Assists)
	assert.Equal(t, 24, len(p.Deaths))
	assert.Equal(t, int64(12334), p.Damage)
	assert.Equal(t, 468, int(math.Round(p.DamagePerMin())))
	assert.Equal(t, 2.3333333333333335, p.KAD())
	assert.Equal(t, 1.7916666666666667, p.KD())
	assert.Equal(t, int64(7542), p.DamageTaken)
	assert.Equal(t, 286.16779758554617, p.DamageTakenPerMin())
	assert.Equal(t, 37, p.Packs())
	assert.Equal(t, 5, p.Captures)

	pSniper := ls.Players[steamid.SID64(76561198023989090)]
	assert.Equal(t, 21, pSniper.HeadShots)

	pSpy := ls.Players[steamid.SID64(76561197963552834)]
	assert.Equal(t, 22, pSpy.BackStabs)

	ls.PrintPlayers(sortTeam)
	ls.PrintHealing()

	// Reset
	//config.Get().Set(config.CfgLogsTfCacheDir, baseDir)
}

func TestParseLine(t *testing.T) {
	type msgTest struct {
		Msg          string
		ExpectedType MsgType
	}
	logs := []msgTest{
		{
			`L 07/10/2019 - 23:28:01: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "11") (weapon "syringegun_medic")`,
			damage,
		}, {
			`L 07/10/2019 - 23:29:54: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`,
			damage,
		}, {
			`L 07/10/2019 - 23:28:02: "rad<6><[U:1:57823119]><Red>" triggered "shot_fired" (weapon "syringegun_medic")`,
			shotFired,
		}, {
			`L 07/10/2019 - 23:28:02: "z/<14><[U:1:66656848]><Blue>" triggered "shot_hit" (weapon "blackbox")`,
			shotHit,
		}, {
			`L 07/10/2019 - 23:28:02: "z/<14><[U:1:66656848]><Blue>" triggered "medic_death" against "rad<6><[U:1:57823119]><Red>" (healing "0") (ubercharge "0")`,
			medicDeath,
		}, {
			`L 07/10/2019 - 23:47:32: "SEND HELP<16><[U:1:84528002]><Blue>" triggered "lost_uber_advantage" (time "44")`,
			lostUberAdv,
		}, {
			`L 07/10/2019 - 23:47:34: "g о а т z<13><[U:1:41435165]><Red>" picked up item "ammopack_small"`,
			pickup,
		}, {
			`L 07/10/2019 - 23:47:33: "the lord of the pings<11><[U:1:114143419]><Blue>" spawned as "Scout"`,
			spawnedAs,
		}, {
			`L 07/10/2019 - 23:13:56: "Graba<3><[U:1:95947321]><>" STEAM USERID validated`,
			validated,
		}, {
			`L 07/10/2019 - 23:15:20: "wonszu #LANsilesia2019<8><[U:1:60952177]><>" connected, address "0.0.0.0:51378"`,
			connected,
		}, {
			`L 07/11/2019 - 00:49:19: "AMP_T<55><[U:1:163893616]><Blue>" disconnected (reason "AMP_T timed out")`,
			disconnected,
		}, {
			`L 07/10/2019 - 23:15:33: "wonszu #LANsilesia2019<8><[U:1:60952177]><>" entered the game`,
			entered,
		}, {
			`L 07/10/2019 - 23:16:05: "Kwq<9><[U:1:96748980]><Unassigned>" joined team "Blue"`,
			joinedTeam,
		}, {
			`L 07/10/2019 - 23:16:05: "Kwq<9><[U:1:96748980]><Blue>" changed role to "soldier"`,
			changeClass,
		}, {
			`L 07/10/2019 - 23:16:39: "Kwq<9><[U:1:96748980]><Blue>" committed suicide with "world" (attacker_position "-1435 -1965 518")`,
			suicide,
		}, {
			`L 07/10/2019 - 23:26:36: "thaZu.pl<4><[U:1:79473044]><Spectator>" say " 811 ms : Kwq"`,
			say,
		}, {
			`L 07/10/2019 - 23:26:36: "thaZu.pl<4><[U:1:79473044]><Spectator>" say_team " 811 ms : Kwq"`,
			sayTeam,
		}, {
			`L 07/10/2019 - 23:26:43: "Kwq<9><[U:1:96748980]><Blue>" triggered "empty_uber"`,
			emptyUber,
		}, {
			`L 07/10/2019 - 23:47:32: "SEND HELP<16><[U:1:84528002]><Blue>" triggered "lost_uber_advantage" (time "44")`,
			lostUberAdv,
		}, {
			`L 07/10/2019 - 23:47:52: "Graba<3><[U:1:95947321]><Blue>" triggered "medic_death" against "wonder<7><[U:1:34284979]><Red>" (healing "3218") (ubercharge "0")`,
			medicDeath,
		}, {
			`L 07/10/2019 - 23:47:52: "wonder<7><[U:1:34284979]><Red>" triggered "medic_death_ex" (uberpct "32")`,
			medicDeathEx,
		}, {
			`L 07/10/2019 - 23:50:32: "stan FIN_SLAYER<10><[U:1:127171744]><Red>" triggered "revenge" against "17<17><[U:1:156985751]><Blue>" (assist "1")`,
			revenge,
		}, {
			`L 07/11/2019 - 00:48:29: "defa<49><[U:1:129337538]><Red>" triggered "revenge" against "AlesKee<59><[U:1:206838965]><Blue>"`,
			revenge,
		}, {
			`L 07/10/2019 - 23:50:32: "rad<6><[U:1:57823119]><Red>" killed "17<17><[U:1:156985751]><Blue>" with "quake_rl" (attacker_position "-1688 -2242 795") (victim_position "-1666 -2536 690")`,
			killed,
		}, {
			`L 07/10/2019 - 23:50:32: "stan FIN_SLAYER<10><[U:1:127171744]><Red>" triggered "kill assist" against "17<17><[U:1:156985751]><Blue>" (assister_position "-1080 -1752 723") (attacker_position "-1688 -2242 795") (victim_position "-1666 -2536 690")`,
			killAssist,
		}, {
			`L 07/11/2019 - 00:11:30: "kartka<15><[U:1:130519691]><Red>" triggered "domination" against "17<17><[U:1:156985751]><Blue>" (assist "1")`,
			domination,
		}, {
			`L 07/11/2019 - 00:11:38: Team "Red" final score "3" with "6" players`,
			wTeamFinalScore,
		}, {
			`L 07/11/2019 - 00:11:28: Team "Red" current score "3" with "6" players`,
			wTeamScore,
		}, {
			`L 07/11/2019 - 00:11:28: World triggered "Round_Win" (winner "Red")`,
			wRoundWin,
		}, {
			`L 07/11/2019 - 00:11:28: World triggered "Round_Length" (seconds "325.86")`,
			wRoundLen,
		}, {
			`L 07/11/2019 - 00:11:38: World triggered "Game_Over" reason "Reached Time Limit"`,
			wGameOver,
		}, {
			`L 07/11/2019 - 00:11:04: "wonder<7><[U:1:34284979]><Red>" triggered "chargeready"`,
			chargeReady,
		}, {
			`L 07/11/2019 - 00:11:11: "wonder<7><[U:1:34284979]><Red>" triggered "chargedeployed" (medigun "medigun")`,
			chargeDeployed,
		}, {
			`L 07/11/2019 - 00:11:18: "wonder<7><[U:1:34284979]><Red>" triggered "chargeended" (duration "7.5")`,
			chargeEnded,
		}, {
			`L 07/11/2019 - 00:11:19: "wonder<7><[U:1:34284979]><Red>" triggered "healed" against "stanFIN_SLAYER<10><[U:1:127171744]><Red>" (healing "16")`,
			healed,
		}, {
			`L 07/11/2019 - 00:09:55: "wonder<7><[U:1:34284979]><Red>" triggered "player_extinguished" against "rad<6><[U:1:57823119]><Red>" with "tf_weapon_medigun" (attacker_position "1907 2554 611") (victim_position "1728 2457 576")`,
			extinguished,
		}, {
			`L 10/25/2019 - 12:14:45: "von<16><[U:1:181030438]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "-1689 -2062 59")`,
			builtObject,
		}, {
			`L 10/25/2019 - 12:16:01: "von<16><[U:1:181030438]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1822 -616 2")`,
			carryObject,
		}, {
			`L 10/25/2019 - 12:15:27: "Sedna<9><[U:1:160531776]><Red>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "1976 630 265")`,
			dropObject,
		}, {
			`L 10/25/2019 - 12:19:36: "Kodyn<19><[U:1:439767837]><Blue>" triggered "killedobject" (object "OBJ_DISPENSER") (weapon "tf_projectile_rocket") (objectowner "Sedna<9><[U:1:160531776]><Red>") (attacker_position "-359 -111 528")`,
			killedObject,
		}, {
			`triggered "killedobject" (object "OBJ_SENTRYGUN") (objectowner "Rambosaur<34><[U:1:66415434]><Blue>") (assist "1") (assister_position "77 233 0") (attacker_position "-415 218 0")`,
			killedObject,
		},
		{
			`L 10/25/2019 - 12:19:46: "SCOTTY T<27><[U:1:97282856]><Blue>" triggered "first_heal_after_spawn" (time "1.6")`,
			firstHealAfterSpawn,
		}, {
			`L 10/25/2019 - 12:21:49: "Andyroo<20><[U:1:229954190]><Blue>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "1168 -374 720")`,
			builtObject,
		}, {
			`L 10/25/2019 - 12:20:53: "von<16><[U:1:181030438]><Blue>" triggered "object_detonated" (object "OBJ_TELEPORTER") (position "470 1326 576")`,
			detonatedObject,
		}, {
			`L 07/11/2019 - 00:28:37: "Detoed<43><[U:1:93656154]><Blue>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-266 343 0")`,
			captureBlocked,
		}, {
			`L 07/11/2019 - 00:38:41: Team "Blue" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "3") (player1 "AustinN<48><[U:1:167925837]><Blue>") (position1 "99 97 7") (player2 "STiNGHAN<51><[U:1:63723362]><Blue>") (position2 "-105 118 5") (player3 "FTH<54><[U:1:106022087]><Blue>") (position3 "-162 -125 0")`,
			pointCaptured,
		}, {
			`L 07/11/2019 - 00:27:17: "Houston<46><[U:1:96048647]><Red>" killed "STiNGHAN<51><[U:1:63723362]><Blue>" with "big_earner" (customkill "backstab") (attacker_position "747 661 208") (victim_position "701 608 208")`,
			killedCustom,
		}, {
			`L 07/11/2019 - 00:26:29: "HLPugsTV<45><his possible><unknown>" changed role to "undefined"`,
			skipped,
		}, {
			`L 07/11/2019 - 00:50:12: "AMP_T<64><[U:1:163893616]><unknown>" spawned as "undefined"`,
			skipped,
		}, {
			`L 10/27/2019 - 23:53:58: World triggered "Game_Paused"`,
			wPaused,
		}, {
			`L 10/27/2019 - 23:53:38: World triggered "Game_Unpaused"`,
			wUnpaused,
		},
	}
	for _, line := range logs {
		_, msgType := parseLine(line.Msg)
		assert.Equal(t, line.ExpectedType, msgType, line.Msg)
	}
}
