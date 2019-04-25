package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

//                ____      __  __                   ___
//    _________  / / /     / /_/ /_  ___        ____/ (_)_______
//   / ___/ __ \/ / /_____/ __/ __ \/ _ \______/ __  / / ___/ _ \
//  / /  / /_/ / / /_____/ /_/ / / /  __/_____/ /_/ / / /__/  __/
// /_/   \____/_/_/      \__/_/ /_/\___/      \__,_/_/\___/\___/

func rollTheDiceInit() {
}

func rollTheDice(message string) (response string, sendToDM bool) {
	var err error

	// rolls that are sent back
	var rolls []int

	rand.Seed(time.Now().UTC().UnixNano())

	log.Printf("roll the dice")
	// Example !roll 1d6
	// Example 50d10
	dice := strings.Split(message, "d")
	rollCount, err := strconv.Atoi(dice[0])
	if err != nil {
		response = fmt.Sprintf("bad format on the dice count")
		sendToDM = false
	}

	dieValue, err := strconv.Atoi(dice[1])
	if err != nil {
		response = fmt.Sprintf("bad format on the dice value")
		sendToDM = false
	}

	switch dieValue {
	case 4, 6, 8, 10, 12, 20, 100:
		log.Printf("good die value")
	default:
		response = fmt.Sprintf("dice are limited to 4,6,8,10,12,20, and 100 sided die")
		sendToDM = false
	}

	if rollCount > 10 {
		response = fmt.Sprintf("rolls are limited to 10 at a time")
		sendToDM = false
	}

	log.Printf("rolling a %d sided die %d times", dieValue, rollCount)
	for i := 0; i < rollCount; i++ {
		rolls = append(rolls, rand.Intn(dieValue)+1)
	}

	for rtdi, val := range rolls {
		response = response + strconv.Itoa(val)
		if rtdi == len(rolls)-2 {
			response = response + ", and "
		} else if rtdi != len(rolls)-1 {
			response = response + ", "
		}
	}

	response = response + fmt.Sprintf(" for a total of %d", total(rolls))

	return
}

func total(dice []int) (total int) {
	for _, die := range dice {
		total = total + die
	}

	return
}
