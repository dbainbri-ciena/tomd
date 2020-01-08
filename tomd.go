package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Attendee struct {
	Name string
}

type Item struct {
	By   string
	Text string
}

type Action struct {
	Who  string
	Text string
	Due  string
}

type Topic struct {
	StartTime   time.Time
	EndTime     time.Time
	Name        string
	Items       []*Item
	Actions     []*Action
	Decisions   []*Item
	NonCommands []*NonCommand
}

type Unknown struct {
	When time.Time
	Who  string
	Text string
}

type NonCommand struct {
	When time.Time
	Who  string
	Text string
}

type Meeting struct {
	Name       string
	StartTime  *time.Time
	ChatOffset time.Duration
	EndTime    *time.Time
	Attendees  []*Attendee
	Topics     []*Topic
	Unknowns   []*Unknown
}

var dumpNonCommands = flag.Bool("non", false, "include the dump of non commands")

func dump(meeting *Meeting) {
	allactions := []*Action{}
	alldecisions := []*Item{}
	allnoncommands := []*NonCommand{}
	fmt.Printf("# %s started on %s\n", meeting.Name, meeting.StartTime)

	fmt.Println("\n## Attendance")

	sort.Slice(meeting.Attendees, func(i, j int) bool {
		return strings.Compare(meeting.Attendees[i].Name,
			meeting.Attendees[j].Name) < 0
	})

	for _, a := range meeting.Attendees {
		fmt.Printf("- %s\n", a.Name)
	}

	for _, t := range meeting.Topics {
		fmt.Printf("\n## Topic: %s (%s)\n", t.Name, t.EndTime.Sub(t.StartTime))
		for _, i := range t.Items {
			fmt.Printf("- %s\n", i.Text)
		}
		if len(t.Actions) > 0 {
			fmt.Println("\n### Actions")
			for _, a := range t.Actions {
				allactions = append(allactions, a)
				fmt.Printf("- %s(%s): %s\n", a.Who, a.Due, a.Text)
			}
		}
		if len(t.Decisions) > 0 {
			fmt.Println("\n### Decisions")
			for _, d := range t.Decisions {
				alldecisions = append(alldecisions, d)
				fmt.Printf("- %s\n", d.Text)
			}
		}
		if *dumpNonCommands && len(t.NonCommands) > 0 {
			fmt.Println("\n### Non Commands")
			for _, u := range t.NonCommands {
				allnoncommands = append(allnoncommands, u)
				fmt.Printf("- %s(%s): %s\n", u.Who, u.When, u.Text)
			}
		}
	}

	if len(allactions) > 0 {
		fmt.Println("\n# All Actions")
		for _, a := range allactions {
			fmt.Printf("- %s(%s): %s\n", a.Who, a.Due, a.Text)
		}
	}

	if len(alldecisions) > 0 {
		fmt.Println("\n# All Decisions")
		for _, d := range alldecisions {
			fmt.Printf("- %s\n", d.Text)
		}
	}

	if len(meeting.Unknowns) > 0 {
		fmt.Println("\n# Unknown Commands")
		for _, u := range meeting.Unknowns {
			fmt.Printf("- %s(%s): %s\n", u.Who, u.When, u.Text)
		}
	}

	if *dumpNonCommands && len(allactions) > 0 {
		fmt.Println("\n# All Non Commands")
		for _, a := range allnoncommands {
			fmt.Printf("- %s(%s): %s\n", a.Who, a.When, a.Text)
		}
	}

	fmt.Printf("\n# Meeting ended at %s\n", meeting.EndTime)
}

func timeOffset(base *time.Time, baseOffset time.Duration, offset string) time.Time {
	parts := strings.Split(offset, ":")
	dur, err := time.ParseDuration(fmt.Sprintf("%sh%sm%ss", parts[0], parts[1], parts[2]))

	if err != nil || base == nil {
		return time.Now().Add(dur)
	}
	return base.Add(dur).Add(-baseOffset)
}

func main() {
	flag.Parse()
	meeting := Meeting{
		Name:      "not specified",
		StartTime: nil,
		EndTime:   nil,
	}
	var currentTopic *Topic = nil

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		parts := regexp.MustCompile("\t+").Split(strings.TrimSpace(scanner.Text()), 3)
		if len(parts) != 3 {
			continue
		}
		when := parts[0]
		who := parts[1]
		if strings.HasSuffix(who, ":") {
			who = who[:len(who)-1]
		}
		text := parts[2]

		cmd := strings.SplitN(text, " ", 2)
		if len(cmd) < 1 {
			continue
		}

		switch strings.TrimSpace(cmd[0]) {
		default:
			ts := timeOffset(meeting.StartTime, meeting.ChatOffset, when)
			if strings.HasPrefix(parts[2], "@") {
				meeting.Unknowns = append(meeting.Unknowns,
					&Unknown{
						When: ts,
						Who:  who,
						Text: text,
					})
			} else {
				if currentTopic == nil {
					currentTopic = &Topic{
						StartTime: timeOffset(meeting.StartTime, meeting.ChatOffset, when),
						Name:      "not specified",
					}
					meeting.Topics = append(meeting.Topics, currentTopic)
				}
				currentTopic.NonCommands = append(currentTopic.NonCommands,
					&NonCommand{
						When: ts,
						Who:  who,
						Text: text,
					})
			}
		case "@here":
			if len(cmd) == 1 {
				meeting.Attendees = append(meeting.Attendees,
					&Attendee{
						Name: who,
					})
			} else {
				meeting.Attendees = append(meeting.Attendees,
					&Attendee{
						Name: cmd[1],
					})
			}
		case "@startmeeting":
			mparts := strings.SplitN(cmd[1], " ", 2)
			if len(mparts) != 2 {
				meeting.Name = cmd[1]
			} else {
				st, err := time.Parse("2006-01-02T15:04MST", mparts[0])
				if err != nil {
					panic(err)
					meeting.StartTime = nil
					meeting.Name = cmd[1]
				} else {
					meeting.StartTime = &st
					meeting.Name = mparts[1]
				}
			}

			// Capture the offset time of the meeting start
			oparts := strings.Split(when, ":")
			odur, err := time.ParseDuration(fmt.Sprintf("%sh%sm%ss", oparts[0], oparts[1], oparts[2]))
			if err == nil {
				meeting.ChatOffset = odur
			}
		case "@endmeeting":
			ts := timeOffset(meeting.StartTime, meeting.ChatOffset, when)
			if currentTopic != nil {
				currentTopic.EndTime = ts
			}
			endtime := ts
			meeting.EndTime = &endtime
		case "@topic":
			ts := timeOffset(meeting.StartTime, meeting.ChatOffset, when)
			if currentTopic != nil {
				currentTopic.EndTime = ts
			}
			currentTopic = &Topic{
				StartTime: ts,
				Name:      cmd[1],
			}
			meeting.Topics = append(meeting.Topics, currentTopic)
		case "@item":
			if currentTopic == nil {
				currentTopic = &Topic{
					StartTime: timeOffset(meeting.StartTime, meeting.ChatOffset, when),
					Name:      "not specified",
				}
				meeting.Topics = append(meeting.Topics, currentTopic)
			}
			currentTopic.Items = append(currentTopic.Items,
				&Item{
					Text: cmd[1],
				})
		case "@decision":
			if currentTopic == nil {
				currentTopic = &Topic{
					StartTime: timeOffset(meeting.StartTime, meeting.ChatOffset, when),
					Name:      "not specified",
				}
				meeting.Topics = append(meeting.Topics, currentTopic)
			}
			currentTopic.Decisions = append(currentTopic.Decisions,
				&Item{
					Text: cmd[1],
				})
		case "@action":
			if currentTopic == nil {
				currentTopic = &Topic{
					StartTime: timeOffset(meeting.StartTime, meeting.ChatOffset, when),
					Name:      "not specified",
				}
				meeting.Topics = append(meeting.Topics, currentTopic)
			}
			aparts := strings.SplitN(cmd[1], " ", 3)
			action := &Action{}
			if len(aparts) >= 1 {
				action.Who = aparts[0]
			}
			if len(aparts) >= 2 {
				action.Due = aparts[1]
			}
			if len(aparts) >= 3 {
				action.Text = aparts[2]
			}
			currentTopic.Actions = append(currentTopic.Actions, action)

		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	dump(&meeting)
}
