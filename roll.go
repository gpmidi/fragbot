package main

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//                ____      __  __                   ___
//    _________  / / /     / /_/ /_  ___        ____/ (_)_______
//   / ___/ __ \/ / /_____/ __/ __ \/ _ \______/ __  / / ___/ _ \
//  / /  / /_/ / / /_____/ /_/ / / /  __/_____/ /_/ / / /__/  __/
// /_/   \____/_/_/      \__/_/ /_/\___/      \__,_/_/\___/\___/

var (
	wanderingInfo wanderingDamage
)

type rtdInfo struct {
	ChannelID string `json:"channel_id,omitempty"`
	Sides     []int  `json:"sides,omitempty"`
}

type wanderingDamage struct {
	LimbLoss        wandering `json:"limb_loss"`
	WanderingDamage wandering `json:"wandering_damage"`
	RandomDamage    wandering `json:"random_damage"`
}

type wandering struct {
	Roll  wanderingRoll    `json:"roll"`
	Table []wanderingTable `json:"table"`
}

type wanderingRoll struct {
	Dice  int `json:"dice"`
	Value int `json:"value"`
}

type wanderingTable struct {
	Outcome rollOutcome   `json:"outcome,omitempty"`
	Result  string        `json:"result"`
	Roll    wanderingRoll `json:"roll,omitempty"`
	Limb    bool          `json:"limb,omitempty"`
	Wander  bool          `json:"wander,omitempty"`
	Random  bool          `json:"random,omitempty"`
	Damage  bool          `json:"damage,omitempty"`
}

type rollOutcome struct {
	Exact int              `json:"exact,omitempty"`
	Range rollOutcomeRange `json:"range,omitempty"`
}

type rollOutcomeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func rollTheDiceInit() {
	var err error

	log.Printf("loading wandering damage info")
	err = loadInfo("wandering.json", &wanderingInfo)
	if err != nil {
		log.Fatalf("there was an issue reading the wandering file\n")
	}
	log.Printf("wandering damage info loaded")
}

func rollTheDice(message string) (response string, sendToDM bool) {
	var err error

	// a users proficiency
	var proficiency int

	log.Printf("roll the dice")
	// Example !roll 1d6+2
	validID, err := regexp.Compile(`(\d+)\s?d\s?(\d+)\s?(?:(\+|\-)\s?(\d*))?(?:\s?(?:x\s?)(\d*)|)`)
	if err != nil {
		log.Printf("There was an error compiling the regex for the roll command")
		return
	}

	dieInfo := validID.FindStringSubmatch(message)

	if len(dieInfo) == 0 {
		return
	}

	rollCount, err := strconv.Atoi(dieInfo[1])
	if err != nil {
		log.Printf("There was an error converting the number of rolls")
	}

	dieValue, err := strconv.Atoi(dieInfo[2])
	if err != nil {
		log.Printf("There was an error converting the number of sides")
	}

	if dieInfo[4] != "" {
		proficiency, err = strconv.Atoi(dieInfo[4])
		if err != nil {
			log.Printf("There was an error converting proficiency")
		}
	}

	if !hasElem(chn.RTD.Sides, dieValue) {
		response = fmt.Sprintf("Only dice with %s sides are supported.", arrayToString(chn.RTD.Sides))
		return
	}

	if rollCount > 10 {
		response = fmt.Sprintf("rolls are limited to 10 at a time")
		return
	}

	if dieInfo[5] != "" {
		log.Printf("rolling %s sets", dieInfo[5])
		multiRoll, err = strconv.Atoi(dieInfo[5])
		if err != nil {
			log.Printf("There was an error converting the number of rolls")
		}
		response = fmt.Sprintf("I have rolled %d sets of rolls for you coming out with \n", multiRoll)
	}

	if multiRoll > 5 {
		response = fmt.Sprintf("Sorry I only support up to 5 sets of rolls.")
		return
	}

	for i := 1; i <= multiRoll; i++ {
		response = response + rollDie(dieInfo[3], dieValue, rollCount, proficiency)
	}

	return
}

func roll(rollCount int, dieValue int) (rolls []int) {
	for i := 0; i < rollCount; i++ {
		rolls = append(rolls, rand.Intn(dieValue)+1)
	}

	return
}

func rollDie(addSub string, dieValue, rollCount, proficiency int) (response string) {
	// strings that are sent back
	var prettyRolls string
	var profString string

	log.Printf("rolling a %d sided die %d times", dieValue, rollCount)
	allRolls := roll(rollCount, dieValue)
	prettyRolls = arrayToString(allRolls)

	rollTotal := total(allRolls)

	log.Printf("%d", rollTotal)

	if addSub == "" {
		log.Printf("No profeciency was added to the roll")
	} else {
		if addSub == "+" {
			log.Printf("Adding %d to the roll", proficiency)
			rollTotal = rollTotal + proficiency
			profString = fmt.Sprintf("adding %d ", proficiency)
		} else if addSub == "-" {
			log.Printf("subtracting %d to the roll", proficiency)
			rollTotal = rollTotal - proficiency
			profString = fmt.Sprintf("subtracting %d ", proficiency)
		} else {

		}
	}

	response = fmt.Sprintf("I have rolled %s %sfor a total of %d \n", prettyRolls, profString, rollTotal)

	return
}

func rollWandering() (response string, sendToDM bool, reroll bool) {

	var outcome int
	var damage int

	rolls := roll(wanderingInfo.WanderingDamage.Roll.Dice, wanderingInfo.WanderingDamage.Roll.Value)

	log.Printf("These are the rolls '%d'", rolls)

	outcome = total(rolls)

	log.Printf("This is the outcome '%d'", outcome)

	// this should never happen. If it does let me know...
	if outcome == 0 {
		log.Printf("If you ever log this line please open a github issue...")
		return
	}

	for _, value := range wanderingInfo.WanderingDamage.Table {
		if value.Outcome.Exact == outcome || between(value.Outcome.Range.Min, value.Outcome.Range.Max, outcome) {
			response = value.Result
			if value.Limb {
				log.Printf("rolling for limb loss")
				response = response + rollLimbLoss()
			} else if value.Random {
				log.Printf("rolling on the random damage table")
			} else if value.Damage {
				log.Printf("rolling for damage")
				rolls = roll(value.Roll.Dice, value.Roll.Value)
				damage = total(rolls)
				response = strings.Replace(value.Result, "&damage&", strconv.Itoa(damage), -1)
			} else if value.Wander {
				log.Printf("rolling on the wandering table again")
				reroll = value.Wander
			}
		}
	}

	log.Printf("%s", response)

	return
}

func rollLimbLoss() (result string) {
	var outcome int

	rolls := roll(wanderingInfo.LimbLoss.Roll.Dice, wanderingInfo.LimbLoss.Roll.Value)

	outcome = total(rolls)

	for _, value := range wanderingInfo.LimbLoss.Table {
		if value.Outcome.Exact == 0 {
		} else if value.Outcome.Exact == outcome {
			result = value.Result
		} else if value.Outcome.Range.Max >= outcome || outcome >= value.Outcome.Range.Min {
			result = value.Result
		}
	}

	return
}

func rollRandom() (result string) {
	var outcome int

	rolls := roll(wanderingInfo.LimbLoss.Roll.Dice, wanderingInfo.LimbLoss.Roll.Value)

	outcome = total(rolls)

	for _, value := range wanderingInfo.LimbLoss.Table {
		if value.Outcome.Exact == 0 {
		} else if value.Outcome.Exact == outcome {
			result = value.Result
		} else if value.Outcome.Range.Max >= outcome || outcome >= value.Outcome.Range.Min {
			result = value.Result
		}
	}

	return
}

func roll(rollCount int, dieValue int) (rolls []int) {
	rand.Seed(time.Now().UTC().UnixNano())

	for i := 0; i < rollCount; i++ {
		rolls = append(rolls, rand.Intn(dieValue)+1)
	}

	return
}

func arrayToString(intArray []int) (pretty string) {
	for rtdi, val := range intArray {
		pretty = pretty + strconv.Itoa(val)
		if rtdi == len(intArray)-2 {
			pretty = pretty + ", and "
		} else if rtdi != len(intArray)-1 {
			pretty = pretty + ", "
		}
	}

	return
}

func between(min, max, num int) (isBetween bool) {
	if max >= num && num >= min {
		isBetween = true
	}

	return
}

// if array has an element
func hasElem(array interface{}, elem interface{}) bool {
	arrV := reflect.ValueOf(array)

	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {
			if arrV.Index(i).Interface() == elem {
				return true
			}
		}
	}

	return false
}
