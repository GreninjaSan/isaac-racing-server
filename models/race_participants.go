package models

/*
	Imports
*/

import (
	"database/sql"
)

/*
	Data types
*/

type RaceParticipants struct{}

/*
	"race_participants" table functions
*/

func (*RaceParticipants) GetCurrentRaces(username string) ([]Race, error) {
	// Get a list of the non-finished races that the user is currently in
	rows, err := db.Query(`
		SELECT races.id, races.status
		FROM race_participants
			JOIN races ON race_participants.race_id = races.id
		WHERE race_participants.user_id = (SELECT id FROM users WHERE username = ?)
			AND races.status != 'finished'
		ORDER BY races.id
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over the races
	var raceList []Race
	for rows.Next() {
		var race Race
		err := rows.Scan(&race.ID, &race.Status)
		if err != nil {
			return nil, err
		}

		// Append this race to the slice
		raceList = append(raceList, race)
	}

	return raceList, nil
}

func (*RaceParticipants) GetNotStartedRaces(userID int) ([]int, error) {
	// Get a list of the non-finished and non-started races that the user is currently in
	rows, err := db.Query(`
		SELECT races.id
		FROM race_participants
			JOIN races ON race_participants.race_id = races.id
		WHERE race_participants.user_id = ?
			AND races.status != 'finished'
			AND races.status != 'in progress'
		ORDER BY races.id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over the races
	var raceIDs []int
	for rows.Next() {
		var raceID int
		err := rows.Scan(&raceID)
		if err != nil {
			return nil, err
		}

		// Append this race to the slice
		raceIDs = append(raceIDs, raceID)
	}

	return raceIDs, nil
}

func (*RaceParticipants) GetFinishedRaces(username string) ([]Race, error) {
	// Get a list of the finished races for this user
	rows, err := db.Query(`
		SELECT races.id, races.format
		FROM race_participants
			JOIN races ON race_participants.race_id = races.id
		WHERE race_participants.user_id = (SELECT id FROM users WHERE username = ?)
			AND race_participants.status = 'finished'
		ORDER BY race_participants.datetime_finished
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over the races
	var raceList []Race
	for rows.Next() {
		var race Race
		err := rows.Scan(&race.ID, &race.Ruleset.Format)
		if err != nil {
			return nil, err
		}

		// Append this race to the slice
		raceList = append(raceList, race)
	}

	return raceList, nil
}

func (*RaceParticipants) GetRacerList(raceID int) ([]Racer, error) {
	// Get the people in this race
	rows, err := db.Query(`
		SELECT
			users.username,
			race_participants.status,
			race_participants.datetime_joined,
			race_participants.datetime_finished,
			race_participants.place,
			race_participants.place_mid,
			race_participants.comment,
			race_participants.floor_num,
			race_participants.floor_arrived,
			users.stream_url,
			users.twitch_bot_enabled,
			users.twitch_bot_delay
		FROM race_participants
			JOIN users ON users.id = race_participants.user_id
		WHERE race_participants.race_id = ?
		ORDER BY race_participants.id
	`, raceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// We have to initialize this way to avoid sending a null on an empty array: https://danott.co/posts/json-marshalling-empty-slices-to-empty-arrays-in-go.html
	racerList := make([]Racer, 0)
	for rows.Next() {
		var racer Racer
		err := rows.Scan(
			&racer.Name,
			&racer.Status,
			&racer.DatetimeJoined,
			&racer.DatetimeFinished,
			&racer.Place,
			&racer.PlaceMid,
			&racer.Comment,
			&racer.FloorNum,
			&racer.FloorArrived,
			&racer.StreamURL,
			&racer.TwitchBotEnabled,
			&racer.TwitchBotDelay,
		)
		if err != nil {
			return nil, err
		}

		// Add it to the list
		racer.Items = make([]Item, 0) // We want this to be an empty array instead of null
		racerList = append(racerList, racer)
	}

	// Get the items for the people in this race
	rows, err = db.Query(`
		SELECT users.username, race_participant_items.item_id, race_participant_items.floor_num
		FROM race_participants
			JOIN users ON users.id = race_participants.user_id
			JOIN race_participant_items ON race_participant_items.race_participant_id = race_participants.id
		WHERE race_participants.race_id = ?
		ORDER BY race_participants.id
	`, raceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// For each item that we found, append it to the appropriate place in the racerList object
	for rows.Next() {
		var username string
		var item Item
		err := rows.Scan(&username, &item.ID, &item.FloorNum)
		if err != nil {
			return nil, err
		}

		// Find the racer in the racerList object
		for i, racer := range racerList {
			if racer.Name == username {
				racerList[i].Items = append(racer.Items, item)
				break
			}
		}
	}

	return racerList, nil
}

func (*RaceParticipants) GetRacerNames(raceID int) ([]string, error) {
	// Get only the names of the people in this race
	rows, err := db.Query(`
		SELECT users.username
		FROM race_participants
			JOIN users ON users.id = race_participants.user_id
		WHERE race_participants.race_id = ?
		ORDER BY race_participants.id
	`, raceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var racerNames []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		racerNames = append(racerNames, name)
	}

	return racerNames, nil
}

func (*RaceParticipants) GetStatus(username string, raceID int) (string, error) {
	// Get the racing status of this person
	var status string
	err := db.QueryRow(`
		SELECT status
		FROM race_participants
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND race_id = ?
	`, username, raceID).Scan(&status)
	if err != nil {
		return status, err
	} else {
		return status, nil
	}
}

func (*RaceParticipants) GetCurrentPlace(raceID int) (int, error) {
	// Get the place of the last person that finished so far
	var place int
	err := db.QueryRow(`
		SELECT MAX(place)
		FROM race_participants
		WHERE race_id = ?
	`, raceID).Scan(&place)
	if err != nil {
		return place, err
	} else {
		return place, nil
	}
}

func (*RaceParticipants) GetFloor(userID int, raceID int) (int, int, error) {
	// Check to see what floor this user is currently on
	var floorNum int
	var stageType int
	err := db.QueryRow(`
		SELECT floor_num, stage_type
		FROM race_participants
		WHERE user_id = ? AND race_id = ?
	`, userID, raceID).Scan(&floorNum, &stageType)
	if err != nil {
		return floorNum, stageType, err
	} else {
		return floorNum, stageType, nil
	}
}

func (*RaceParticipants) GetPeopleLeft(raceID int) (int, error) {
	// Get the number of people left in the race
	var peopleLeft int
	err := db.QueryRow(`
		SELECT COUNT(id)
		FROM race_participants
		WHERE race_id = ? AND status = 'racing'
	`, raceID).Scan(&peopleLeft)
	if err != nil {
		return peopleLeft, err
	} else {
		return peopleLeft, nil
	}
}

func (*RaceParticipants) CheckInRace1(userID int, raceID int) (bool, error) {
	// Playing or observing
	var id int
	err := db.QueryRow(`
		SELECT id
		FROM race_participants
		WHERE user_id = ? AND race_id = ?
	`, userID, raceID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (*RaceParticipants) CheckInRace2(userID int, raceID int) (bool, error) {
	// ONLY playing (not observing)
	var id int
	err := db.QueryRow(`
		SELECT id
		FROM race_participants
		WHERE user_id = ? AND race_id = ? AND status != 'observing'
	`, userID, raceID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (*RaceParticipants) CheckStatus(userID int, raceID int, correctStatus string) (bool, error) {
	// Check to see if the user has this status
	var status string
	err := db.QueryRow(`
		SELECT status
		FROM race_participants
		WHERE user_id = ? AND race_id = ?
	`, userID, raceID).Scan(&status)
	if err != nil {
		return false, err
	} else if status != correctStatus {
		return false, nil
	} else {
		return true, nil
	}
}

func (*RaceParticipants) CheckAllStatus(raceID int, correctStatus string) (bool, error) {
	// Check to see if everyone in the race has this status
	rows, err := db.Query(`
		SELECT status
		FROM race_participants
		WHERE race_id = ?
	`, raceID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	// Iterate over the racers
	sameStatus := true
	for rows.Next() {
		var status string
		err := rows.Scan(&status)
		if err != nil {
			return false, err
		} else if status != correctStatus {
			sameStatus = false
			break
		}
	}

	return sameStatus, nil
}

func (*RaceParticipants) CheckStillRacing(raceID int) (bool, error) {
	// Check if anyone in the race is still racing
	var count int
	err := db.QueryRow(`
		SELECT COUNT(id) as count
		FROM race_participants
		WHERE race_id = ? AND status == 'racing'
	`, raceID).Scan(&count)
	if err != nil {
		return false, err
	} else if count == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func (*RaceParticipants) SetStatus(username string, raceID int, status string) error {
	// Set the new status for the user
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET status = ?
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND race_id = ?
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(status, username, raceID)
	if err != nil {
		return err
	}

	return nil
}

func (*RaceParticipants) SetAllStatus(raceID int, status string) error {
	// Update the status for everyone in this race
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET status = ?
		WHERE race_id = ?
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(status, raceID)
	if err != nil {
		return err
	}

	return nil
}

func (*RaceParticipants) SetDatetimeFinished(username string, raceID int, datetime int) error {
	// Set the finish time for the user
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET datetime_finished = ?
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND race_id = ?
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(datetime, username, raceID)
	if err != nil {
		return err
	}

	return nil
}

func (*RaceParticipants) SetPlace(username string, raceID int, place int) error {
	// Set the mid-race place and the final place for the user
	// (although it shouldn't matter what their mid-race place is at this point)
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET place = ?, place_mid = ?
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND race_id = ?
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(place, place, username, raceID)
	if err != nil {
		return err
	}

	return nil
}

func (*RaceParticipants) SetPlaceMid(username string, raceID int, placeMid int) error {
	// Set the (mid-race) place for the user
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET place_mid = ?
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND race_id = ?
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(placeMid, username, raceID)
	if err != nil {
		return err
	}

	return nil
}

func (*RaceParticipants) SetComment(userID int, raceID int, comment string) error {
	// Set the comment for the user
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET comment = ?
		WHERE user_id = ?
			AND race_id = ?
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(comment, userID, raceID)
	if err != nil {
		return err
	}

	return nil
}

func (*RaceParticipants) SetFloor(userID int, raceID int, floorNum int, stageType int) (int, error) {
	// Set the floor for the user
	currentTime := int(makeTimestamp())
	stmt, err := db.Prepare(`
		UPDATE race_participants
		SET floor_num = ?, stage_type = ?, floor_arrived = ?
		WHERE user_id = ?
			AND race_id = ?
	`)
	if err != nil {
		return currentTime, err
	}
	_, err = stmt.Exec(floorNum, stageType, currentTime, userID, raceID)
	if err != nil {
		return currentTime, err
	}

	return currentTime, nil
}

func (*RaceParticipants) Insert(userID int, raceID int) error {
	// Add the user to the participants list for that race
	stmt, err := db.Prepare(`
		INSERT INTO race_participants (user_id, race_id, datetime_joined)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(userID, raceID, makeTimestamp())
	if err != nil {
		return err
	}

	return nil
}

func (self *RaceParticipants) Delete(username string, raceID int) error {
	// Remove the user from the participants list for the respective race
	if stmt, err := db.Prepare(`
		DELETE FROM race_participants
		WHERE user_id = (SELECT id FROM users WHERE username = ?)
			AND race_id = ?
	`); err != nil {
		return err
	} else {
		_, err := stmt.Exec(username, raceID)
		if err != nil {
			return err
		}
	}

	// Check to see if anyone is still in this race
	if racerNames, err := self.GetRacerNames(raceID); err != nil {
		return err
	} else if len(racerNames) == 0 {
		// Automatically close the race
		if stmt, err := db.Prepare(`
			DELETE FROM races
			WHERE id = ?
		`); err != nil {
			return err
		} else {
			_, err := stmt.Exec(raceID)
			if err != nil {
				return err
			}
		}
	} else {
		// Check to see if this user was the captain
		var userID int
		if err := db.QueryRow(`
			SELECT id
			FROM users
			WHERE username = ?
		`, username).Scan(&userID); err != nil {
			return err
		}
		var captain int
		if err := db.QueryRow(`
			SELECT captain
			FROM races
			WHERE id = ?
		`, raceID).Scan(&captain); err != nil {
			return err
		}
		if userID == captain {
			// Change the captain to someone else
			stmt, err := db.Prepare(`
				UPDATE races
				SET captain = (SELECT user_id from race_participants WHERE race_id = ? ORDER BY id LIMIT 1)
				WHERE id = ?
			`)
			if err != nil {
				return err
			}
			_, err = stmt.Exec(raceID, raceID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
