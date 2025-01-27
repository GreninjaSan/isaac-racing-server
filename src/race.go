package server

import (
	"strconv"
	"time"

	"github.com/Zamiell/isaac-racing-server/models"
)

/*
	Data structures
*/

// Used to track the current races in memory
type Race struct {
	ID              int
	Name            string
	Status          RaceStatus
	Ruleset         Ruleset
	Captain         string
	Password        string
	SoundPlayed     bool
	DatetimeCreated int64
	DatetimeStarted int64
	Racers          map[string]*Racer // Indexed by racer name
}

type Ruleset struct {
	Ranked              bool       `json:"ranked"`
	Solo                bool       `json:"solo"`
	Format              RaceFormat `json:"format"`
	Character           string     `json:"character"`
	CharacterRandom     bool       `json:"characterRandom"`
	Goal                RaceGoal   `json:"goal"`
	StartingBuild       int        `json:"startingBuild"`
	StartingBuildRandom bool       `json:"startingBuildRandom"`
	Seed                string     `json:"seed"`
	Difficulty          string     `json:"difficulty"`
}

/*
	Race object methods
*/

// Get the place that someone would be if they finished the race right now
func (race *Race) GetCurrentPlace() int {
	currentPlace := 0
	for _, racer := range race.Racers {
		if racer.Place > currentPlace {
			currentPlace = racer.Place
		}
	}

	return currentPlace + 1
}

func (race *Race) GetLastPlace() int {
	lastPlace := len(race.Racers)
	for _, racer := range race.Racers {
		if racer.Status == RacerStatusQuit || racer.Status == RacerStatusDisqualified {
			lastPlace--
		}
	}

	return lastPlace
}

// Check to see if a race is ready to start, and if so, start it
// (called from the "websocketRaceReady" and "websocketRaceLeave" functions)
func (race *Race) CheckStart() {
	// Check to see if there is only 1 person in the race
	if len(race.Racers) == 1 && !race.Ruleset.Solo {
		return
	}

	// Check if everyone is ready
	for _, racer := range race.Racers {
		if racer.Status != RacerStatusReady {
			return
		}
	}

	race.Start()
}

func (race *Race) SetStatus(status RaceStatus) {
	race.Status = status

	for _, s := range websocketSessions {
		type RaceSetStatusMessage struct {
			ID     int        `json:"id"`
			Status RaceStatus `json:"status"`
		}
		websocketEmit(s, "raceSetStatus", &RaceSetStatusMessage{
			ID:     race.ID,
			Status: race.Status,
		})
	}
}

func (race *Race) SetRacerStatus(username string, status RacerStatus) {
	racer := race.Racers[username]
	racer.Status = status

	for racerName := range race.Racers {
		// Not all racers may be online during a race
		if s, ok := websocketSessions[racerName]; ok {
			type RacerSetStatusMessage struct {
				ID      int         `json:"id"`
				Name    string      `json:"name"`
				Status  RacerStatus `json:"status"`
				Place   int         `json:"place"`
				RunTime int64       `json:"runTime"`
			}
			websocketEmit(s, "racerSetStatus", &RacerSetStatusMessage{
				ID:      race.ID,
				Name:    username,
				Status:  status,
				Place:   racer.Place,
				RunTime: racer.RunTime,
			})
		}
	}
}

// Recalculate everyone's mid-race places
func (race *Race) SetAllPlaceMid() {
	// Get the place that someone would be if they finished the race right now
	currentPlace := race.GetCurrentPlace()
	lastPlace := race.GetLastPlace()
	if debug {
		logger.Debug("Recalculating mid-race places.")
		logger.Debug("currentPlace:", currentPlace)
		logger.Debug("lastPlace:", lastPlace)
	}

	for _, racer := range race.Racers {
		racer.PlaceMidOld = racer.PlaceMid
		if debug {
			logger.Debug("Set PlaceMidOld for "+racer.Name+" to:", racer.PlaceMidOld)
		}
	}

	for _, racer := range race.Racers {
		if racer.Status != RacerStatusRacing {
			// We don't need to calculate the mid-race place of someone who already finished or quit
			if debug {
				logger.Debug("Skipping " + racer.Name + "since they already finished or quit.")
			}
			continue
		}

		racerOnRepentanceFloor := isRepentanceStageType(racer.StageType)

		if racer.FloorNum == 1 && racer.CharacterNum == 1 && !racerOnRepentanceFloor && !racer.BackwardsPath {
			// Mid-race places are not calculated until racers get to the second floor
			racer.PlaceMid = lastPlace
			if debug {
				logger.Debug("Skipping " + racer.Name + "since they are on the first floor finished or quit.")
			}
			continue
		}

		racer.PlaceMid = currentPlace

		// Find racers that should be ahead of us
		for _, racer2 := range race.Racers {
			// Skip ourselves
			if racer2.ID == racer.ID {
				continue
			}

			// We don't count people who finished or quit since our starting point is on
			// "currentPlace"
			if racer2.Status != RacerStatusRacing {
				continue
			}

			// If they are on a lower character than us, we must be ahead of them
			if racer2.CharacterNum < racer.CharacterNum {
				continue
			}

			// If they are on a higher character than us, we must be behind them
			if racer2.CharacterNum > racer.CharacterNum {
				racer.PlaceMid++
				continue
			}

			adjustedFloorNumUs := getAdjustedFloorNum(racer)
			adjustedFloorNumThem := getAdjustedFloorNum(racer2)

			// Only account for "Backwards Path" logic on races with specific goals
			if race.Ruleset.Goal == RaceGoalBeast || race.Ruleset.Goal == RaceGoalCustom {
				// If they are not on the backwards path, and we are on the backwards path,
				// we must be ahead of them
				if !racer2.BackwardsPath && racer.BackwardsPath {
					continue
				}

				// If they are on the backwards path, and we are not on the backwards path,
				// we must be behind them
				if racer2.BackwardsPath && !racer.BackwardsPath {
					racer.PlaceMid++
					continue
				}

				// If we are both on the backwards path, then floor logic is inverted
				if racer2.BackwardsPath && racer.BackwardsPath {
					// If they are on a higher floor than us, we must be ahead of them
					// (since we are going downwards)
					if adjustedFloorNumThem > adjustedFloorNumUs {
						continue
					}

					// If they are on a lower floor than us, we must be behind them
					// (since we are going downwards)
					if adjustedFloorNumThem < adjustedFloorNumUs {
						racer.PlaceMid++
						continue
					}
				}
			}

			// If they are on a lower floor than us, we must be ahead of them
			if adjustedFloorNumThem < adjustedFloorNumUs {
				continue
			}

			// If they are on a higher floor than us, we must be behind them
			if adjustedFloorNumThem > adjustedFloorNumUs {
				racer.PlaceMid++
				continue
			}

			// If they are on the same floor and they arrived after us, we must be ahead of them
			if racer2.DatetimeArrivedFloor > racer.DatetimeArrivedFloor {
				continue
			}

			// If they are on the same floor and they arrived before us, we must be behind them
			if racer2.DatetimeArrivedFloor <= racer.DatetimeArrivedFloor {
				racer.PlaceMid++
				continue
			}

			logger.Errorf(
				"The \"SetAllPlaceMid()\" function failed to find a condition to sort player \"%s\" and \"%s\".",
				racer.Name,
				racer2.Name,
			)
		}
	}

	for _, racer := range race.Racers {
		if racer.PlaceMidOld != racer.PlaceMid {
			race.SendAllPlaceMid(racer.Name, racer.PlaceMid)
		}
	}
}

const (
	StageTypeOriginal = iota
	StageTypeWoTL
	StageTypeAfterbirth
	StageTypeGreedMode
	StageTypeRepentance
	StagetypeRepentanceB
)

// Account for Repentance floors being offset by 1
func getAdjustedFloorNum(racer *Racer) int {
	if isRepentanceStageType(racer.StageType) {
		return racer.FloorNum + 1
	}

	// If the player reach Home, we need to tell the server that we're on a lower floor than B1 (since we're going backwards)
	if racer.FloorNum == 13 {
		return 0
	}

	return racer.FloorNum
}

func isRepentanceStageType(stageType int) bool {
	return stageType == StageTypeRepentance || stageType == StagetypeRepentanceB
}

func (race *Race) SendAllPlaceMid(username string, placeMid int) {
	for racerName := range race.Racers {
		// Not all racers may be online during a race
		if s, ok := websocketSessions[racerName]; ok {
			type RacerSetPlaceMidMessage struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				PlaceMid int    `json:"placeMid"`
			}
			websocketEmit(s, "racerSetPlaceMid", &RacerSetPlaceMidMessage{
				ID:       race.ID,
				Name:     username,
				PlaceMid: placeMid,
			})
		}
	}
}

// Called from the "CheckStart" function
func (race *Race) Start() {
	var secondsToWait int
	if race.Ruleset.Solo {
		secondsToWait = 3
	} else {
		secondsToWait = 10
	}

	// Log the race starting
	logger.Info("Race "+strconv.Itoa(race.ID)+" starting in", secondsToWait, "seconds.")

	// Change the status for this race to "starting"
	race.SetStatus("starting")

	// Send everyone in the race a message specifying exactly when it will start
	for racerName := range race.Racers {
		// A racer might go offline the moment before it starts, so check just in case
		if s, ok := websocketSessions[racerName]; ok {
			websocketEmit(s, "raceStart", &RaceStartMessage{
				ID:            race.ID,
				SecondsToWait: secondsToWait,
			})
		}
	}

	// Make the Twitch bot announce that the race is starting in 10 seconds
	twitchRaceStart(race)

	// Return for now and do more things in 10 seconds
	go race.Start2()
}

func (race *Race) Start2() {
	// Sleep 3 or 10 seconds
	var sleepSeconds int
	if race.Ruleset.Solo {
		sleepSeconds = 3
	} else {
		sleepSeconds = 10
	}
	time.Sleep(time.Duration(sleepSeconds) * time.Second)

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()
	defer commandMutex.Unlock()

	// Log the race starting
	logger.Info("Race", race.ID, "started with", len(race.Racers), "participants.")

	race.SetStatus(RaceStatusInProgress)
	race.DatetimeStarted = getTimestamp()

	numRacers := len(race.Racers)
	for _, racer := range race.Racers {
		racer.Status = RacerStatusRacing
		racer.PlaceMid = numRacers // Make everyone tied for last place
	}

	// Return for now and do more things later on when it is time to check to see if the race has
	// been going for too long
	go race.Start3()
}

func (race *Race) Start3() {
	if race.Ruleset.Format == RaceFormatCustom {
		// We need to make the timeout longer to accommodate multi-character speedrun races
		time.Sleep(4 * time.Hour)
	} else {
		time.Sleep(30 * time.Minute)
	}

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()
	defer commandMutex.Unlock()

	// Find out if the race is finished
	// (we remove finished races from the "races" map)
	if _, ok := races[race.ID]; !ok {
		return
	}

	// Force the remaining racers to quit
	for _, racer := range race.Racers {
		if racer.Status != RacerStatusRacing {
			continue
		}

		logger.Info("Forcing racer \"" + racer.Name + "\" to quit since the race time limit has been reached.")

		d := &IncomingWebsocketData{
			Command: "race.Start3",
			ID:      race.ID,
			v: &models.SessionValues{
				Username: racer.Name,
			},
		}
		websocketRaceQuit(nil, d)
	}
}

func (race *Race) CheckFinish() {
	for _, racer := range race.Racers {
		if racer.Status == RacerStatusRacing {
			return
		}
	}

	race.Finish()
}

func (race *Race) Finish() {
	// Log the race finishing
	logger.Info("Race " + strconv.Itoa(race.ID) + " finished.")

	// Let everyone know it ended
	race.SetStatus("finished")

	// Remove it from the map
	delete(races, race.ID)

	// Write it to the database
	databaseRace := &models.Race{
		ID:              race.ID,
		Name:            race.Name,
		Ranked:          race.Ruleset.Ranked,
		Solo:            race.Ruleset.Solo,
		Format:          string(race.Ruleset.Format),
		Character:       race.Ruleset.Character,
		Goal:            string(race.Ruleset.Goal),
		Difficulty:      race.Ruleset.Difficulty,
		StartingBuild:   race.Ruleset.StartingBuild,
		Seed:            race.Ruleset.Seed,
		Captain:         race.Captain,
		DatetimeStarted: race.DatetimeStarted,
	}
	if err := db.Races.Finish(databaseRace); err != nil {
		logger.Error("Failed to write race #"+strconv.Itoa(race.ID)+" to the database:", err)
		return
	}

	for _, racer := range race.Racers {
		databaseRacer := &models.Racer{
			ID:               racer.ID,
			DatetimeJoined:   racer.DatetimeJoined,
			Seed:             racer.Seed,
			StartingItem:     racer.StartingItem,
			Place:            racer.Place,
			DatetimeFinished: racer.DatetimeFinished,
			RunTime:          racer.RunTime,
			Comment:          racer.Comment,
		}
		if err := db.RaceParticipants.Insert(race.ID, databaseRacer); err != nil {
			logger.Error("Failed to write the RaceParticipants row for \""+race.Name+"\" to the database:", err)
			return
		}

		for _, item := range racer.Items {
			if err := db.RaceParticipantItems.Insert(
				racer.ID,
				race.ID,
				item.ID,
				item.FloorNum,
				item.StageType,
				item.DatetimeAcquired,
			); err != nil {
				logger.Error("Failed to write the RaceParticipantItems row for \""+race.Name+"\" to the database:", err)
				return
			}
		}

		for _, room := range racer.Rooms {
			if err := db.RaceParticipantRooms.Insert(
				racer.ID,
				race.ID,
				room.ID,
				room.FloorNum,
				room.StageType,
				room.DatetimeArrived,
			); err != nil {
				logger.Error("Failed to write the RaceParticipantRooms row for \""+race.Name+"\" to the database:", err)
				return
			}
		}
	}

	if race.Ruleset.Solo {
		if race.Ruleset.Ranked {
			leaderboardUpdateRankedSolo(race)
		}
	} else {
		if race.Ruleset.Format == RaceFormatUnseeded ||
			race.Ruleset.Format == RaceFormatSeeded ||
			race.Ruleset.Format == RaceFormatDiversity {

			leaderboardUpdateTrueSkill(race)
		}
	}
}
