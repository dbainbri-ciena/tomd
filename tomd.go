package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
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
	Name    string
	Items   []*Item
	Actions []*Action
}

type Unknown struct {
	When time.Time
	Who  string
	Text string
}

type Meeting struct {
	Name      string
	StartTime *time.Time
	EndTime   *time.Time
	Attendees []*Attendee
	Topics    []*Topic
	Unknowns  []*Unknown
}

func dump(meeting *Meeting) {
	allactions := []*Action{}
	fmt.Printf("# %s started on %s\n", meeting.Name, meeting.StartTime)

	fmt.Println("\n## Attendance")
	for _, a := range meeting.Attendees {
		fmt.Printf("- %s\n", a.Name)
	}

	for _, t := range meeting.Topics {
		fmt.Printf("\n## Topic: %s\n", t.Name)
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
	}

	if len(allactions) > 0 {
		fmt.Println("\n# All Actions")
		for _, a := range allactions {
			fmt.Printf("- %s(%s): %s\n", a.Who, a.Due, a.Text)
		}
	}

	if len(meeting.Unknowns) > 0 {
		fmt.Println("\n# Unknown Commands")
		for _, u := range meeting.Unknowns {
			fmt.Printf("- %s(%s): %s\n", u.Who, u.When, u.Text)
		}
	}

	fmt.Printf("\n# Meeting ended at %s\n", meeting.EndTime)
}

func timeOffset(base *time.Time, offset string) time.Time {
	parts := strings.Split(offset, ":")
	dur, err := time.ParseDuration(fmt.Sprintf("%sh%sm%ss", parts[0], parts[1], parts[2]))

	if err != nil || base == nil {
		return time.Now().Add(dur)
	}
	return base.Add(dur)
}

func main() {
	meeting := Meeting{
		Name:      "not specified",
		StartTime: nil,
		EndTime:   nil,
	}
	var currentTopic *Topic = nil

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		parts := regexp.MustCompile("\t+").Split(scanner.Text(), 3)
		if len(parts) != 3 || !strings.HasPrefix(parts[2], "@") {
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

		switch cmd[0] {
		default:
			ts := timeOffset(meeting.StartTime, when)
			meeting.Unknowns = append(meeting.Unknowns,
				&Unknown{
					When: ts,
					Who:  who,
					Text: text,
				})
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
		case "@meeting":
			mparts := strings.SplitN(cmd[1], " ", 2)
			if len(mparts) != 2 {
				meeting.Name = cmd[1]
			} else {
				st, err := time.Parse("2006-01-02T15:04MST", mparts[0])
				if err != nil {
					meeting.StartTime = nil
					meeting.Name = cmd[1]
				} else {
					meeting.StartTime = &st
					meeting.Name = mparts[1]
				}
			}
		case "@endmeeting":
			endtime := timeOffset(meeting.StartTime, when)
			meeting.EndTime = &endtime
		case "@topic":
			currentTopic = &Topic{
				Name: cmd[1],
			}
			meeting.Topics = append(meeting.Topics, currentTopic)
		case "@item":
			if currentTopic == nil {
				currentTopic = &Topic{
					Name: "not specified",
				}
				meeting.Topics = append(meeting.Topics, currentTopic)
			}
			currentTopic.Items = append(currentTopic.Items,
				&Item{
					Text: cmd[1],
				})
		case "@action":
			if currentTopic == nil {
				currentTopic = &Topic{
					Name: "not specified",
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
