package main

import (
	"fmt"
	"strings"
)

// Category of a sample. Unmatched samples are neutral.
type Category string

const (
	Productive  Category = "productive"
	Neutral     Category = "neutral"
	Distracting Category = "distracting"
)

// Rule classifies a sample by case-insensitive substring match on its app
// name or window title. Title rules are evaluated before app rules (chrome
// on "youtube" is distracting even if chrome itself is neutral); within
// each group, first match wins.
type Rule struct {
	Match    string   `json:"match"`   // "app" or "title"
	Pattern  string   `json:"pattern"` // substring, case-insensitive
	Category Category `json:"category"`
}

func defaultRules() []Rule {
	return []Rule{
		{Match: "title", Pattern: "youtube", Category: Distracting},
		{Match: "title", Pattern: "netflix", Category: Distracting},
		{Match: "title", Pattern: "twitter", Category: Distracting},
		{Match: "title", Pattern: "x.com", Category: Distracting},
		{Match: "title", Pattern: "stack overflow", Category: Productive},
		{Match: "title", Pattern: "pkg.go.dev", Category: Productive},
		{Match: "title", Pattern: "github", Category: Productive},
		{Match: "app", Pattern: "code", Category: Productive},
		{Match: "app", Pattern: "windowsterminal", Category: Productive},
		{Match: "app", Pattern: "goland", Category: Productive},
	}
}

// classifySample returns the category of a sample: title rules first, then
// app rules, first match wins, unmatched is neutral.
func classifySample(s Sample, rules []Rule) Category {
	app := strings.ToLower(s.App)
	title := strings.ToLower(s.Title)
	for _, pass := range []string{"title", "app"} {
		for _, r := range rules {
			if r.Match != pass {
				continue
			}
			subject := app
			if r.Match == "title" {
				subject = title
			}
			if subject != "" && strings.Contains(subject, strings.ToLower(r.Pattern)) {
				return r.Category
			}
		}
	}
	return Neutral
}

// classify implements `lacheisis classify [<app|title> <pattern> <category>]`:
// with no args it lists rules, otherwise it appends one.
func classify(cfg Config, args []string) error {
	if len(args) == 0 {
		if len(cfg.Rules) == 0 {
			fmt.Println("no rules set")
			return nil
		}
		for _, r := range cfg.Rules {
			fmt.Printf("  %-6s %-24q → %s\n", r.Match, r.Pattern, r.Category)
		}
		return nil
	}
	if len(args) != 3 {
		return fmt.Errorf("usage: lacheisis classify <app|title> <pattern> <category>")
	}
	match, pattern, cat := args[0], args[1], Category(args[2])
	if match != "app" && match != "title" {
		return fmt.Errorf("match must be app or title, got %q", match)
	}
	if cat != Productive && cat != Neutral && cat != Distracting {
		return fmt.Errorf("category must be productive, neutral or distracting, got %q", cat)
	}
	cfg.Rules = append(cfg.Rules, Rule{Match: match, Pattern: pattern, Category: cat})
	if err := saveConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("rule added: %s %q → %s\n", match, pattern, cat)
	return nil
}
