package main

import (
	"fmt"
	"log"
	"regexp"
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

type lfgInfo struct {
	ChannelID string   `json:"channel_id,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
}

type platforms struct {
	Platforms []platformInfo `json:"platforms"`
}

type platformInfo struct {
	Name  string     `json:"platform_type"`
	Games []gameInfo `json:"game,omitempty"`
}

type gameInfo struct {
	Title   string   `json:"game_title"`
	Players []string `json:"game_players,omitempty"`
}

type players struct {
	Players []playerInfo `json:"players,omitempty"`
}

type playerInfo struct {
	DiscordID string `json:"discord_id"`
	Game      string `json:"game,omitempty"`
	Name      string `json:"player_name"`
	Platform  string `json:"platform,omitempty"`
	Until     int64  `json:"look_until"`
}

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

	// vars for lfg
	var game string
	var platform string
	var timeInt int

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

		saveInfo("lfg_platforms.json", platfms) // Need to capture errors
		saveInfo("lfg_users.json", users)       // Need to capture errors
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
				gameTitle := platGame.Title
				if len(platGame.Title) > 22 {
					log.Printf("Truncating string")
					gameTitle = platGame.Title[:21]
				}
				codeBlock.Message = append(codeBlock.Message, fmt.Sprintf("%s%s%s", pad.Left("", 12, " "), pad.Right(gameTitle, 22, " "), strings.Join(platGame.Players, ", ")))
			}
		}

		response = response + codeBlock.Header
		response = response + strings.Join(codeBlock.Message, "\n")
		response = response + codeBlock.Footer

		return response, false
	}

	validID, err := regexp.Compile(`([a-zA-Z]+\d?|[a-zA-Z]+(?:[[:punct:]])+(?:\s?)|[a-zA-Z]+(?:\s?))+(?:\s?)(\d+|$)`)
	if err != nil {
		log.Printf("There was an error compiling the regex for the lfg command")
		return
	}

	lfgQuery := validID.FindStringSubmatch(message)

	if lfgQuery[2] == "" {
		game = strings.TrimSuffix(lfgQuery[0], fmt.Sprintf(" %s", lfgQuery[1]))
		platform = lfgQuery[1]
		timeInt = 60
	} else {
		game = strings.TrimSuffix(lfgQuery[0], fmt.Sprintf(" %s %s", lfgQuery[1], lfgQuery[2]))
		platform = lfgQuery[1]
		log.Printf("setting timeInt to %s", lfgQuery[2])
		timeInt, err = strconv.Atoi(lfgQuery[2])
		if err != nil {
			return fmt.Sprintf("bad format on the time to wait"), false
		}
	}

	if game == "" {
		return fmt.Sprintf("no game specified"), false
	}

	// if no game or no platform specified
	if platform == "" {
		return fmt.Sprintf("no platform specified that is recognized, I support the following platforms: %s", strings.Join(chn.LFG.Platforms, ", ")), false
	}

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

	saveInfo("lfg_platforms.json", platfms) // Need to capture errors
	saveInfo("lfg_users.json", users)       // Need to capture errors

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

		saveInfo("lfg_platforms.json", platfms) // Need to capture errors
		saveInfo("lfg_users.json", users)       // Need to capture errors
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
