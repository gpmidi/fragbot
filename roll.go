package main

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
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
	var prettyRolls string

	// a users proficiency
	var proficiency int
	var profString string

	rand.Seed(time.Now().UTC().UnixNano())

	log.Printf("roll the dice")
	// Example !roll 1d6+2
	validID, err := regexp.Compile(`(\d+)\s?d\s?(\d+)\s?(?:(\+|\-)\s?(\d*))?`)
	if err != nil {
		log.Printf("There was an error compiling the regex for the roll command")
		return
	}

	dieInfo := validID.FindStringSubmatch(message)

	dieValue, err := strconv.Atoi(dieInfo[2])
	if err != nil {
		log.Printf("There was an error converting the number of sides")
	}

	rollCount, err := strconv.Atoi(dieInfo[1])
	if err != nil {
		log.Printf("There was an error converting the number of rolls")
	}

	if dieInfo[4] != "" {
		proficiency, err = strconv.Atoi(dieInfo[4])
		if err != nil {
			log.Printf("There was an error converting proficiency")
		}
	}

	switch dieValue {
	case 4, 6, 8, 10, 12, 20, 100:
		log.Printf("good die value")
	default:
		response = fmt.Sprintf("dice are limited to 4,6,8,10,12,20, and 100 sided die")
	}

	if rollCount > 10 {
		response = fmt.Sprintf("rolls are limited to 10 at a time")
	}

	log.Printf("rolling a %d sided die %d times", dieValue, rollCount)
	allRolls := roll(rollCount, dieValue)
	for rtdi, val := range allRolls {
		prettyRolls = prettyRolls + strconv.Itoa(val)
		if rtdi == len(allRolls)-2 {
			prettyRolls = prettyRolls + ", and "
		} else if rtdi != len(allRolls)-1 {
			prettyRolls = prettyRolls + ", "
		}
	}

	rollTotal := total(allRolls)

	log.Printf("%d", rollTotal)

	if dieInfo[3] == "" || dieInfo[4] == "" {
		log.Printf("No profeciency was added to the roll")
	} else {
		if dieInfo[3] == "+" {
			log.Printf("Adding %d to the roll", proficiency)
			rollTotal = rollTotal + proficiency
			profString = fmt.Sprintf("adding %d ", proficiency)
		} else if dieInfo[3] == "-" {
			log.Printf("subtracting %d to the roll", proficiency)
			rollTotal = rollTotal - proficiency
			profString = fmt.Sprintf("subtracting %d ", proficiency)
		} else {

		}
	}

	response = fmt.Sprintf("I have rolled %s %sfor a total of %d", prettyRolls, profString, rollTotal)

	return
}

func roll(rollCount int, dieValue int) (rolls []int) {
	for i := 0; i < rollCount; i++ {
		rolls = append(rolls, rand.Intn(dieValue)+1)
	}

	return
}

func total(dice []int) (total int) {
	for _, die := range dice {
		total = total + die
	}

	return
}
