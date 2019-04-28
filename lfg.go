package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/willf/pad"
)

//     __            __   _                   ____
//    / /___  ____  / /__(_)___  ____ _      / __/___  _____      ____ __________  __  ______
//   / / __ \/ __ \/ //_/ / __ \/ __ `/_____/ /_/ __ \/ ___/_____/ __ `/ ___/ __ \/ / / / __ \
//  / / /_/ / /_/ / ,< / / / / / /_/ /_____/ __/ /_/ / /  /_____/ /_/ / /  / /_/ / /_/ / /_/ /
// /_/\____/\____/_/|_/_/_/ /_/\__, /     /_/  \____/_/         \__, /_/   \____/\__,_/ .___/
//                            /____/                           /____/                /_/

func lookingForGroupInit() {
	var err error

	log.Printf("loading users")
	err = loadInfo("lfg_users.json", &users)
	if err != nil {
		log.Fatalf("there was an issue reading the users file\n")
	}
	log.Printf("users loaded")

	log.Printf("loading platform info")
	err = loadPlatforms()
	if err != nil {
		log.Printf("there was an error loading platforms")
	}
	log.Printf("platforms loaded")

	// look for expired users on startup.
	log.Printf("checking for users that are past due")
	lookingForGroupTickJob()
}

func lookingForGroup(message string, authorID string, authorName string) (response string, sendToDM bool) {
	// easier to set an error var now
	var err error

	if message == "leave" {
		log.Printf("removing user from the lfg queue")
		for i, player := range users.Players {
			if player.Name == authorName {
				log.Printf("removing player %s from the queue", player.Name)
				users.Players[i] = users.Players[0]
				users.Players = users.Players[1:]
			}
		}
		lookingForGroupRemovePlatformPlayer(authorName)

		saveInfo("lfg_platforms.json", platfms)
		saveInfo("lfg_users.json", users)
		return "you have left the lfg queue", true
	}

	// user is looking for info about themself
	if message == "me" {
		var playerConf playerInfo
		for _, user := range users.Players {
			if user.Name == authorName {
				playerConf = user
			}
		}

		if playerConf.Name == "" {
			return "You aren't waiting for a game right now.", true
		}

		return fmt.Sprintf("You are waiting for a group to play `%s` for another %d minutes", playerConf.Game, int(time.Until(time.Unix(playerConf.Until, 0)).Minutes())), true
	}

	// user wants a list of games/platofrms and people waiting to play those.
	if message == "list" {
		codeBlock := newCodeBlock()
		codeBlock.Message = append(codeBlock.Message, "platform    game                  players")

		for _, plat := range platfms.Platforms {
			codeBlock.Message = append(codeBlock.Message, fmt.Sprintf("%s", plat.Name))
			if len(plat.Games) != 0 {
			}
			for _, platGame := range plat.Games {
				codeBlock.Message = append(codeBlock.Message, fmt.Sprintf("%s%s%s", pad.Left("", 12, " "), pad.Right(platGame.Title, 22, " "), strings.Join(platGame.Players, ", ")))
			}
		}

		response = response + codeBlock.Header
		response = response + strings.Join(codeBlock.Message, "\n")
		response = response + codeBlock.Footer

		return response, false
	}

	lfgQuery := strings.Split(message, " ")

	// platform from the command
	var platform string
	// game from the command
	var gameArray []string
	var game string

	// time variables because reasons
	var timeStr string
	var timeInt int

	// get full game name before platform
	for i, word := range lfgQuery {
		// loop until you see a platform name
		if contains(chn.LFG.Platforms, word) {
			platform = word
			// set time as next value after platform
			if i == len(lfgQuery)+1 {
				log.Printf("no time sent with message")
			} else {
				timeStr = lfgQuery[i+1]
			}
			break
		}
		gameArray = append(gameArray, word)
	}

	game = strings.Join(gameArray, " ")

	// if no game or no platform specified
	if platform == "" {
		return fmt.Sprintf("no platform specified that is recognized, I support the following platforms: %s", strings.Join(chn.LFG.Platforms, ", ")), false
	}

	if game == "" {
		return fmt.Sprintf("no game specified"), false
	}

	// time stuff
	if timeStr == "" {
		// if no time was specifiec
		log.Printf("no time specified defaulting to 60 minutes")
		timeInt = 60
	} else {
		// set time to lenght specified
		log.Printf("setting timeInt to %s", timeStr)
		timeInt, err = strconv.Atoi(timeStr)
		if err != nil {
			return fmt.Sprintf("bad format on the time to wait"), false
		}
	}

	// log.Printf("%d", timeInt)

	log.Printf("updating player info")
	// player functions
	if len(users.Players) == 0 {
		log.Printf("no users exist appending user")
		users.Players = append(users.Players, playerInfo{
			authorID,
			game,
			authorName,
			platform,
			time.Now().Add(time.Duration(timeInt) * time.Minute).Unix(),
		})
	} else {
		// range over players
		for usi := range users.Players {
			log.Printf("Players i count:%d total player count: %d", usi, len(users.Players)-1)
			// if player exists in the data
			if users.Players[usi].Name == authorName {
				// is game and platform are the same.
				if users.Players[usi].Game == game && users.Players[usi].Platform == platform {
					users.Players[usi].Until = time.Now().Add(time.Duration(timeInt) * time.Minute).Unix()
					break
				} else if users.Players[usi].Game != game || users.Players[usi].Platform != platform {
					log.Printf("updating game to %s and platform to %s", game, platform)
					if len(users.Players) < 1 {
						users.Players[usi] = users.Players[0]
						users.Players = users.Players[1:]
					}
					lookingForGroupRemovePlatformPlayer(authorName)
					users.Players[usi].Game = game
					users.Players[usi].Platform = platform
					users.Players[usi].Until = time.Now().Add(time.Duration(timeInt) * time.Minute).Unix()
					break
				}
				break
				// if the user doesn't exist
			} else if usi == len(users.Players)-1 && users.Players[usi].Name != authorName {
				log.Printf("adding new user to user config")
				playerConf := playerInfo{
					authorID,
					game,
					authorName,
					platform,
					time.Now().Add(time.Duration(timeInt) * time.Minute).Unix(),
				}
				users.Players = append(users.Players, playerConf)
				break
			}
		}
	}

	log.Printf("updating platform info")
	// platform functions
	for pfi := range platfms.Platforms {
		if platfms.Platforms[pfi].Name == platform {
			if len(platfms.Platforms[pfi].Games) == 0 {
				log.Printf("no games exists on this platform appending")
				platfms.Platforms[pfi].Games = append(platfms.Platforms[pfi].Games, gameInfo{
					game,
					[]string{authorName},
				})
				break
			} else {
				for gmi := range platfms.Platforms[pfi].Games {
					if platfms.Platforms[pfi].Games[gmi].Title == game {
						for pli := range platfms.Platforms[pfi].Games[gmi].Players {
							log.Printf("Game i value: %d game total count: %d", pli, len(platfms.Platforms[pfi].Games))
							if platfms.Platforms[pfi].Games[gmi].Players[pli] == authorName {
								break
							}
							// if we get through the array and the player is not found
							if pli == len(platfms.Platforms[pfi].Games[gmi].Players)-1 {
								platfms.Platforms[pfi].Games[gmi].Players = append(platfms.Platforms[pfi].Games[gmi].Players, authorName)
								break
							}
						}
					}

					if gmi == len(platfms.Platforms[pfi].Games)-1 && platfms.Platforms[pfi].Games[gmi].Title != game {
						platfms.Platforms[pfi].Games = append(platfms.Platforms[pfi].Games, gameInfo{
							game,
							[]string{authorName},
						})
						break
					}
				}
			}
		}
	}

	saveInfo("lfg_platforms.json", platfms)
	saveInfo("lfg_users.json", users)

	return response, false
}

func lookingForGroupTickJob() (response string, discordUserID string, send bool) {
	// log.Printf("checking for expiring/expired users")
	for i, user := range users.Players {
		if time.Time.Before(time.Unix(user.Until, 0), time.Now()) {
			log.Printf("user %s is due to be removed", user.Name)
			log.Printf("Dropping user %s due to time passing", user.Name)
			users.Players[i] = users.Players[0]
			users.Players = users.Players[1:]

			lookingForGroupRemovePlatformPlayer(user.Name)
		}
		// after now
		after4 := time.Time.After(time.Unix(user.Until, 0), time.Now().Add(time.Duration(285*time.Second)))
		// before 5 minutes from now
		before5 := time.Time.Before(time.Unix(user.Until, 0), time.Now().Add(time.Duration(300*time.Second)))

		if after4 && before5 {
			log.Printf("user %s has less than 5 minutes remaining", user.Name)
			response = fmt.Sprintf("You have 5 minutes left in the lfg queue for %s", user.Game)
			discordUserID = user.DiscordID
			send = true
		}

		saveInfo("lfg_platforms.json", platfms)
		saveInfo("lfg_users.json", users)
	}

	return
}

func lookingForGroupRemovePlatformPlayer(userName string) {
	for pt, plat := range platfms.Platforms {
		// log.Printf("%s", plat)
		for gm, platGame := range plat.Games {
			// log.Printf("%s", platGame)
			for pl, gamePlayer := range platGame.Players {
				// log.Printf("%s", gamePlayer)
				if gamePlayer == userName {
					platGame.Players[pl] = platGame.Players[0]
					platfms.Platforms[pt].Games[gm].Players = platGame.Players[1:]
					// log.Printf("%s", platGame.Players)
				}
			}
		}
		for gm, platGame := range plat.Games {
			// log.Printf("%s", plat.Games)
			// log.Printf("%s", platGame.Players)
			if len(platGame.Players) == 0 {
				plat.Games[gm] = plat.Games[0]
				platfms.Platforms[pt].Games = plat.Games[1:]
			}
		}
	}
}

func loadPlatforms() error {
	err := loadInfo("lfg_platforms.json", &platfms)
	if err != nil {
		return fmt.Errorf("there was an issue reading the platform file %s", err)
	}

	// log.Printf("%s", platfms.Platforms)

	var platConf platformInfo
	var platArray []string
	for _, platPlat := range platfms.Platforms {
		platArray = append(platArray, platPlat.Name)
	}

	for _, chanPlat := range chn.LFG.Platforms {
		if !strings.Contains(strings.Join(platArray, ", "), chanPlat) {
			// log.Printf("adding platform %s to the lfg_platforms.json", chanPlat)
			platConf.Name = chanPlat
			platfms.Platforms = append(platfms.Platforms, platConf)
		}
	}

	err = saveInfo("lfg_platforms.json", platfms)
	if err != nil {
		return fmt.Errorf("there was an issue updating the config %s", err)
	}
	return nil
}
