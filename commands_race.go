package main

/*
 *  Imports
 */

import (
	"github.com/Zamiell/isaac-racing-server/models"

	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
 *  WebSocket race command functions
 */

func raceCreate(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceCreate"
	userID := conn.UserID
	username := conn.Username
	name := data.Name
	ruleset := data.Ruleset

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate that the race name cannot be empty
	if name == "" {
		name = "-"
	}

	// Validate that the ruleset options cannot be empty
	if ruleset.Type == "" {
		ruleset.Type = "unseeded"
	}
	if ruleset.Character == "" {
		ruleset.Character = "judas"
	}
	if ruleset.Goal == "" {
		ruleset.Goal = "chest"
	}
	if ruleset.Seed == "" {
		ruleset.Seed = "-"
	}

	// Validate the submitted ruleset
	if raceValidateRuleset(conn, data, functionName) == false {
		return
	}

	// Check if this user has started 2 races
	if count, err := db.Races.CaptainCount(userID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	} else if count >= 2 {
		commandMutex.Unlock()
		log.Info("New race request denied; user is captain of ", count, "races.")
		connError(conn, functionName, "To prevent abuse, you are only allowed to create 2 new races at a time.")
		return
	}

	// Check if there are non-finished races with the same name
	if raceWithSameName, err := db.Races.CheckName(name); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	} else if raceWithSameName == true {
		commandMutex.Unlock()
		connError(conn, functionName, "There is already a non-finished race with that name.")
		return
	}

	// Get a seed if necessary
	if (ruleset.Type == "seeded" && ruleset.Seed == "-") ||
		(ruleset.Type == "diversity" && ruleset.Seed == "-") {

		if seed, err := db.Seeds.Get(); err != nil {
			commandMutex.Unlock()
			log.Error("Database error:", err)
			connError(conn, functionName, "Something went wrong. Please contact an administrator.")
			return
		} else {
			ruleset.Seed = seed
		}
	} else if ruleset.Type == "unseeded" || ruleset.Type == "normal" {
		ruleset.Seed = "-"
	}

	// Create the race
	raceID, err := db.Races.Insert(name, ruleset, userID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Add tis user to the participants list for that race
	if err := db.RaceParticipants.Insert(userID, raceID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Send everyone a notification that a new race has been started
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceCreated", &model.Race{
			ID:              raceID,
			Name:            name,
			Status:          "open",
			Ruleset:         ruleset,
			DatetimeCreated: int(time.Now().Unix()),
			Captain:         username,
			Racers:          []string{username},
		})
	}
	connectionMap.RUnlock()

	// Join the user to the channel for that race
	roomJoinSub(conn, "_race_"+strconv.Itoa(raceID))

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceJoin(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceJoin"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race is open
	if raceValidateStatus(conn, data, "open", functionName) == false {
		return
	}

	// Validate that they are not in the race
	if raceValidateOut(conn, data, functionName) == false {
		return
	}

	// Add this user to the participants list for that race
	if err := db.RaceParticipants.Insert(userID, raceID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Join the user to the channel for that race
	roomJoinSub(conn, "_race_"+strconv.Itoa(raceID))

	// Send everyone a notification that the user joined
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceJoined", RaceMessage{raceID, username})
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceLeave(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceLeave"
	username := conn.Username
	raceID := data.ID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race is open
	if raceValidateStatus(conn, data, "open", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Remove this user from the participants list for that race
	if err := db.RaceParticipants.Delete(username, raceID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Disconnect the user from the channel for that race
	roomLeaveSub(conn, "_race_"+strconv.Itoa(raceID))

	// Send everyone a notification that the user left the race
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceLeft", RaceMessage{raceID, username})
	}
	connectionMap.RUnlock()

	// Check to see if the race is ready to start
	raceCheckStart(raceID)

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceReady(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceReady"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race is open
	if raceValidateStatus(conn, data, "open", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that their status is set to "not ready"
	if racerValidateStatus(conn, userID, raceID, "not ready", functionName) == false {
		return
	}

	// Change their status to "ready"
	if racerSetStatus(conn, username, raceID, "ready", functionName) == false {
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user is ready
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			conn.Connection.Emit("racerSetStatus", &RacerSetStatusMessage{raceID, username, "ready"})
		}
	}
	connectionMap.RUnlock()

	// Check to see if the race is ready to start
	raceCheckStart(raceID)

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceUnready(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceUnready"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race is open
	if raceValidateStatus(conn, data, "open", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that their status is set to "ready"
	if racerValidateStatus(conn, userID, raceID, "ready", functionName) == false {
		return
	}

	// Change their status to "not ready"
	if racerSetStatus(conn, username, raceID, "not ready", functionName) == false {
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user is not ready
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			conn.Connection.Emit("racerSetStatus", &RacerSetStatusMessage{raceID, username, "not ready"})
		}
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceRuleset(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceUnready"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID
	ruleset := data.Ruleset

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Get the current ruleset
	currentRuleset, err := db.Races.GetRuleset(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Check to see if anything has changed
	if currentRuleset.Type == ruleset.Type &&
		currentRuleset.Character == ruleset.Character &&
		currentRuleset.Goal == ruleset.Goal &&
		currentRuleset.Seed == ruleset.Seed &&
		currentRuleset.InstantStart == ruleset.InstantStart {

		commandMutex.Unlock()
		connError(conn, functionName, "The race ruleset is already set to those values.")
		return
	}

	// If they didn't specify something, set it to the existing value
	if ruleset.Type == "" {
		ruleset.Type = currentRuleset.Type
	}
	if ruleset.Character == "" {
		ruleset.Character = currentRuleset.Character
	}
	if ruleset.Goal == "" {
		ruleset.Goal = currentRuleset.Goal
	}
	if ruleset.Seed == "" {
		ruleset.Seed = currentRuleset.Seed
	}

	// Validate the submitted ruleset
	if raceValidateRuleset(conn, data, functionName) == false {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race is open
	if raceValidateStatus(conn, data, "open", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that they are the race captain
	if isCaptain, err := db.Races.CheckCaptain(raceID, userID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	} else if isCaptain == false {
		commandMutex.Unlock()
		connError(conn, functionName, "Only the captain of the race can change the ruleset.")
		return
	}

	// Get a seed if necessary
	if (ruleset.Type == "seeded" && currentRuleset.Type != "seeded" && ruleset.Seed == "-") ||
		(ruleset.Type == "diversity" && currentRuleset.Type != "diversity" && ruleset.Seed == "-") {

		if seed, err := db.Seeds.Get(); err != nil {
			commandMutex.Unlock()
			log.Error("Database error:", err)
			connError(conn, functionName, "Something went wrong. Please contact an administrator.")
			return
		} else {
			ruleset.Seed = seed
		}
	} else if ruleset.Type == "unseeded" || ruleset.Type == "normal" {
		ruleset.Seed = "-"
	}

	// Change the ruleset
	if err := db.Races.SetRuleset(raceID, ruleset); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Set everyone's status to "not ready"
	if err := db.RaceParticipants.SetAllStatus(raceID, "not ready"); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Send everyone a notification that the ruleset has changed for this race
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceSetRuleset", RaceSetRulesetMessage{raceID, ruleset})
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceFinish(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceFinish"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race has started
	if raceValidateStatus(conn, data, "in progress", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that their status is set to "racing" status
	if racerValidateStatus(conn, userID, raceID, "racing", functionName) == false {
		return
	}

	// Change their status to "finished"
	if racerSetStatus(conn, username, raceID, "finished", functionName) == false {
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user finished
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			conn.Connection.Emit("racerSetStatus", &RacerSetStatusMessage{raceID, username, "finished"})
		}
	}
	connectionMap.RUnlock()

	// Check to see if the race is ready to finish
	raceCheckFinish(raceID)

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceQuit(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceQuit"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race has started
	if raceValidateStatus(conn, data, "in progress", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that their status is set to "racing" status
	if racerValidateStatus(conn, userID, raceID, "racing", functionName) == false {
		return
	}

	// Change their status to "quit"
	if racerSetStatus(conn, username, raceID, "quit", functionName) == false {
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user quit
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			conn.Connection.Emit("racerSetStatus", &RacerSetStatusMessage{raceID, username, "quit"})
		}
	}
	connectionMap.RUnlock()

	// Check to see if the race is ready to finish
	raceCheckFinish(raceID)

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceComment(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceQuit"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID
	comment := data.Comment

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Strip leading and trailing whitespace from the comment
	comment = strings.TrimSpace(comment)

	// Validate that the comment is not empty
	if comment == "" {
		commandMutex.Unlock()
		connError(conn, functionName, "That is an invalid comment.")
		return
	}

	// Validate that the user is not squelched
	if conn.Squelched == 1 {
		commandMutex.Unlock()
		connError(conn, functionName, "You have been squelched by an administrator, so you cannot submit comments.")
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race has started
	if raceValidateStatus(conn, data, "in progress", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Set their comment in the database
	if err := db.RaceParticipants.SetComment(userID, raceID, comment); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user added or changed their comment
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			conn.Connection.Emit("racerSetComment", &RacerSetCommentMessage{raceID, username, comment})
		}
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceItem(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceItem"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID
	itemID := data.ItemID

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate that the item number is sane
	if itemID < 1 || itemID > 441 { // This will need to be updated once we know the highest item ID in Afterbirth+
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to add an item", itemID, "to their build, but that is a bogus number.")
		connError(conn, functionName, "That is not a valid item ID.")
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race has started
	if raceValidateStatus(conn, data, "in progress", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that their status is set to "racing" status
	if racerValidateStatus(conn, userID, raceID, "racing", functionName) == false {
		return
	}

	// Get their current floor
	floor, err := db.RaceParticipants.GetFloor(userID, raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Add this item to their build
	if err := db.RaceParticipantItems.Insert(userID, raceID, itemID, floor); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user got an item
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			item := model.Item{itemID, floor}
			conn.Connection.Emit("racerAddItem", &RacerAddItemMessage{raceID, username, item})
		}
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

func raceFloor(conn *ExtendedConnection, data *IncomingCommandMessage) {
	// Local variables
	functionName := "raceFloor"
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID
	floor := data.Floor

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Log the received command
	log.Debug("User \""+username+"\" sent a", functionName, "command.")

	// Rate limit all commands
	if commandRateLimit(conn) == true {
		return
	}

	// Validate that the floor is sane
	if floor < 1 || floor > 10 {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to update their floor, but", floor, "is a bogus number.")
		connError(conn, functionName, "That is not a valid floor.")
		return
	}

	// Validate basic things about the race ID
	if raceValidate(conn, data, functionName) == false {
		return
	}

	// Validate that the race has started
	if raceValidateStatus(conn, data, "in progress", functionName) == false {
		return
	}

	// Validate that they are in the race
	if raceValidateIn(conn, data, functionName) == false {
		return
	}

	// Validate that their status is set to "racing" status
	if racerValidateStatus(conn, userID, raceID, "racing", functionName) == false {
		return
	}

	// Set their floor in the database
	if err := db.RaceParticipants.SetFloor(userID, raceID, floor); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return
	}

	// Send success confirmation
	connSuccess(conn, functionName, data)

	// Get the list of racers for this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// Send a notification to all the people in this particular race that the user got to a new floor
	connectionMap.RLock()
	for _, racer := range racerList {
		conn, ok := connectionMap.m[racer.Name]
		if ok == true { // Not all racers may be online during a race
			conn.Connection.Emit("racerSetFloor", &RacerSetFloorMessage{raceID, username, floor})
		}
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

/*
 *  Race subroutines
 */

func raceValidate(conn *ExtendedConnection, data *IncomingCommandMessage, functionName string) bool {
	// Local variables
	username := conn.Username
	raceID := data.ID

	// Validate that the requested race is sane
	if raceID <= 0 {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to call", functionName, "with a bogus ID of "+strconv.Itoa(raceID)+".")
		connError(conn, functionName, "You must provide a valid race number.")
		return false
	}

	// Validate that the requested race exists
	if exists, err := db.Races.Exists(raceID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return false
	} else if exists == false {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to call", functionName, "on race ID "+strconv.Itoa(raceID)+", but it doesn't exist.")
		connError(conn, functionName, "Race ID "+strconv.Itoa(raceID)+" does not exist.")
		return false
	}

	// The user's request seems to be valid
	return true
}

func raceValidateStatus(conn *ExtendedConnection, data *IncomingCommandMessage, status string, functionName string) bool {
	// Local variables
	username := conn.Username
	raceID := data.ID

	// Validate that the race is set to the correct status
	if correctStatus, err := db.Races.CheckStatus(raceID, status); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return false
	} else if correctStatus == false {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to call", functionName, "on race ID "+strconv.Itoa(raceID)+", but race is not set to status \""+status+"\".")
		connError(conn, functionName, "Race ID "+strconv.Itoa(raceID)+" is not set to status \""+status+"\".")
		return false
	}

	// The race is the correct status
	return true
}

func raceValidateRuleset(conn *ExtendedConnection, data *IncomingCommandMessage, functionName string) bool {
	// Local variables
	ruleset := data.Ruleset

	// Validate the ruleset type
	if ruleset.Type != "unseeded" && ruleset.Type != "seeded" && ruleset.Type != "diversity" && ruleset.Type != "vanilla" {
		commandMutex.Unlock()
		connError(conn, functionName, "That is not a valid ruleset.")
		return false
	}

	// Validate the character
	if ruleset.Character != "isaac" &&
		ruleset.Character != "magdalene" &&
		ruleset.Character != "cain" &&
		ruleset.Character != "judas" &&
		ruleset.Character != "blue baby" &&
		ruleset.Character != "eve" &&
		ruleset.Character != "samson" &&
		ruleset.Character != "azazel" &&
		ruleset.Character != "lazarus" &&
		ruleset.Character != "eden" &&
		ruleset.Character != "the lost" &&
		ruleset.Character != "lilith" &&
		ruleset.Character != "keeper" {

		commandMutex.Unlock()
		connError(conn, functionName, "That is not a valid character.")
		return false
	}

	// Validate the goal
	if ruleset.Goal != "chest" && ruleset.Goal != "dark room" && ruleset.Goal != "mega satan" {
		commandMutex.Unlock()
		connError(conn, functionName, "That is not a valid goal.")
		return false
	}

	// Validate the seed
	if ruleset.Seed != "-" {
		ruleset.Seed = strings.ToUpper(ruleset.Seed)
		ruleset.Seed = strings.Replace(ruleset.Seed, " ", "", -1)
		alphanumeric := regexp.MustCompile(`^[A-Z0-9]{8}$`) // Upper case letters or numbers x8
		if alphanumeric.MatchString(ruleset.Seed) == false {
			commandMutex.Unlock()
			connError(conn, functionName, "That is not a valid seed.")
			return false
		}
	}

	// Validate the instant start
	if ruleset.InstantStart < 0 || ruleset.InstantStart > 441 { // This will need to be updated once we know the highest item ID in Afterbirth+
		commandMutex.Unlock()
		connError(conn, functionName, "That is not a valid instant start item.")
		return false
	}

	return true
}

func raceValidateIn(conn *ExtendedConnection, data *IncomingCommandMessage, functionName string) bool {
	// Local variables
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Validate that they are in the race
	if userInRace, err := db.RaceParticipants.CheckInRace(userID, raceID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return false
	} else if userInRace == false {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to call", functionName, "on race ID "+strconv.Itoa(raceID)+", but they are not in that race.")
		connError(conn, functionName, "You are not in race ID "+strconv.Itoa(raceID)+".")
		return false
	}

	// The user is in the race
	return true
}

func raceValidateOut(conn *ExtendedConnection, data *IncomingCommandMessage, functionName string) bool {
	// Local variables
	userID := conn.UserID
	username := conn.Username
	raceID := data.ID

	// Validate that they are not already in the race
	if userInRace, err := db.RaceParticipants.CheckInRace(userID, raceID); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return false
	} else if userInRace == true {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to call", functionName, "on race ID "+strconv.Itoa(raceID)+", but they are already in that race.")
		connError(conn, functionName, "You are already in race ID "+strconv.Itoa(raceID)+".")
		return false
	}

	// The user is not in the race
	return true
}

func racerValidateStatus(conn *ExtendedConnection, userID int, raceID int, status string, functionName string) bool {
	// Local variables
	username := conn.Username

	// Validate that the user is set to the correct status
	if correctStatus, err := db.RaceParticipants.CheckStatus(userID, raceID, status); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return false
	} else if correctStatus == false {
		commandMutex.Unlock()
		log.Warning("User \""+username+"\" attempted to call", functionName, "on race ID "+strconv.Itoa(raceID)+", but they are not set to status \""+status+"\".")
		connError(conn, functionName, "You can only do that if your status is set to \""+status+"\".")
		return false
	}

	// The user has the correct status
	return true
}

func racerSetStatus(conn *ExtendedConnection, username string, raceID int, status string, functionName string) bool {
	// Change the status in the database
	if err := db.RaceParticipants.SetStatus(username, raceID, status); err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		connError(conn, functionName, "Something went wrong. Please contact an administrator.")
		return false
	}

	// The change was successful
	return true
}

// Called after someone disconnects or someone is banned
func raceCheckStartFinish(raceID int) {
	// Get the status of the race
	if status, err := db.Races.GetStatus(raceID); err != nil {
		log.Error("Database error:", err)
		return
	} else if status == "open" {
		raceCheckStart(raceID)
	} else if status == "in progress" {
		raceCheckFinish(raceID)
	}
}

// Check to see if a race is ready to start, and if so, start it
func raceCheckStart(raceID int) {
	// Check if everyone is ready
	if sameStatus, err := db.RaceParticipants.CheckAllStatus(raceID, "ready"); err != nil {
		log.Error("Database error:", err)
		return
	} else if sameStatus == false {
		return
	}

	// Log the race starting
	log.Info("Race " + strconv.Itoa(raceID) + " started.")

	// Change the status for this race to "starting"
	if err := db.Races.SetStatus(raceID, "starting"); err != nil {
		log.Error("Database error:", err)
		return
	}

	// Send everyone a notification that the race is starting soon
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceSetStatus", &RaceSetStatusMessage{raceID, "starting"})
	}
	connectionMap.RUnlock()

	// Get the list of people in this race
	racers, err := db.RaceParticipants.GetRacerNames(raceID)
	if err != nil {
		log.Error("Database error:", err)
		return
	}

	// Send everyone in the race a message describing exactly when it will start
	connectionMap.RLock()
	for _, username := range racers {
		conn, ok := connectionMap.m[username]
		if ok == true {
			conn.Connection.Emit("raceStart", &RaceStartMessage{
				raceID,
				time.Now().Add(10 * time.Second).UnixNano(), // 10 seconds in the future
			})
		} else {
			log.Warning("Failed to send a raceStart message to user \"" + username + "\". This should never happen.")
		}
	}
	connectionMap.RUnlock()

	// Return for now and do more things in 10 seconds
	go raceCheckStart2(raceID)
}

func raceCheckStart2(raceID int) {
	// Sleep 10 seconds
	time.Sleep(10 * time.Second)

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Change the status for this race to "in progress" and set "datetime_started" equal to now
	if err := db.Races.Start(raceID); err != nil {
		log.Error("Database error:", err)
		return
	}

	// Update the status for everyone in the race to "racing"
	if err := db.RaceParticipants.SetAllStatus(raceID, "racing"); err != nil {
		log.Error("Database error:", err)
		return
	}

	// Send everyone a notification that the race is now in progress
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceSetStatus", &RaceSetStatusMessage{raceID, "in progress"})
	}
	connectionMap.RUnlock()

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()

	// Return for now and do more things in 30 minutes
	go raceCheckStart3(raceID)
}

func raceCheckStart3(raceID int) {
	// Sleep 30 minutes
	time.Sleep(30 * time.Minute)

	// Lock the command mutex for the duration of the function to ensure synchronous execution
	commandMutex.Lock()

	// Find out if the race is finished
	if status, err := db.Races.GetStatus(raceID); err != nil {
		log.Error("Database error:", err)
		return
	} else if status == "finished" {
		return
	}

	// The race is still going, so get the list of people still in this race
	racerList, err := db.RaceParticipants.GetRacerList(raceID)
	if err != nil {
		commandMutex.Unlock()
		log.Error("Database error:", err)
		return
	}

	// If any are still racing, force them to quit
	for _, racer := range racerList {
		if racer.Status == "racing" {
			if err := db.RaceParticipants.SetStatus(racer.Name, raceID, "quit"); err != nil {
				log.Error("Database error:", err)
				return
			}
		}
	}

	// Close down the race
	raceCheckFinish(raceID)

	// The command is over, so unlock the command mutex
	commandMutex.Unlock()
}

// Check to see if a rate is ready to finish, and if so, finish it
func raceCheckFinish(raceID int) {
	// Check if anyone is still racing
	if stillRacing, err := db.RaceParticipants.CheckStillRacing(raceID); err != nil {
		log.Error("Database error:", err)
		return
	} else if stillRacing == true {
		return
	}

	// Log the race finishing
	log.Info("Race " + strconv.Itoa(raceID) + " finished.")

	// Finish the race
	if err := db.Races.Finish(raceID); err != nil {
		log.Error("Database error:", err)
		return
	}

	// Send everyone a notification that the race is now finished
	connectionMap.RLock()
	for _, conn := range connectionMap.m {
		conn.Connection.Emit("raceSetStatus", &RaceSetStatusMessage{raceID, "finished"})
	}
	connectionMap.RUnlock()
}
