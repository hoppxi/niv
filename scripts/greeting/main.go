package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type Greeting struct {
	Content string `json:"content"`
}

func getDynamicGreeting(userAtHost string) string {
	hour := time.Now().Hour()
	rand.NewSource(time.Now().UnixNano())

	messages := map[string][]string{
		"morning": {
			"Rise and shine, %s! A fresh morning awaits.",
			"Good morning, %s! Time to seize the day.",
			"Top of the morning, %s! Make today amazing.",
			"Hey %s! May your coffee be strong and your day be bright.",
			"%s, mornings are for fresh starts and big dreams.",
		},
		"afternoon": {
			"Hey %s! Hope your afternoon is full of energy and success.",
			"%s, may your afternoon be productive and refreshing!",
			"Good afternoon, %s! Keep shining.",
			"%s, the afternoon is perfect for small wins and big smiles.",
			"%s, afternoons are for progress and inspiration.",
		},
		"evening": {
			"Good evening, %s! Time to relax and recharge.",
			"%s, hope your evening is calm and peaceful.",
			"Hey %s! Enjoy the evening vibes.",
			"%s, the day is winding down – make the most of your evening!",
			"Evening greetings, %s! Take a moment to reflect on the day.",
		},
		"night": {
			"Good night, %s! Sweet dreams await.",
			"%s, may your night be restful and cozy.",
			"Hey %s! Time to recharge for tomorrow.",
			"%s, hope your night is peaceful and refreshing.",
			"Nighty night, %s! Sleep well and dream big.",
		},
	}

	var selected []string

	switch {
	case hour >= 5 && hour < 12:
		selected = messages["morning"]
	case hour >= 12 && hour < 17:
		selected = messages["afternoon"]
	case hour >= 17 && hour < 21:
		selected = messages["evening"]
	default:
		selected = messages["night"]
	}

	return fmt.Sprintf(selected[rand.Intn(len(selected))], userAtHost)
}

func main() {
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}

	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	userAtHost := fmt.Sprintf("%s@%s", user, host)
	greeting := Greeting{Content: getDynamicGreeting(userAtHost)}

	jsonOutput, _ := json.Marshal(greeting)
	fmt.Println(string(jsonOutput))
}
