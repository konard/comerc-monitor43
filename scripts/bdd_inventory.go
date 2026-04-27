package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type location struct {
	Status string
	File   string
	Line   int
}

type record struct {
	Epic    string
	Feature string
	Status  string
	Text    string
	Sample  string
}

type counter struct {
	Key    string
	Count  int
	Epics  int
	Status string
	Text   string
	Sample string
}

var locationRE = regexp.MustCompile(`^(pending|undefined)\s+(.+):([0-9]+)$`)

func main() {
	input := "out.txt"
	if len(os.Args) > 1 {
		input = os.Args[1]
	}

	locations, err := readLocations(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bdd inventory: %v\n", err)
		os.Exit(1)
	}
	if len(locations) == 0 {
		fmt.Printf("No pending or undefined BDD steps found in %s.\n", input)
		return
	}

	records, err := buildRecords(locations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bdd inventory: %v\n", err)
		os.Exit(1)
	}

	printReport(input, records)
}

func readLocations(input string) ([]location, error) {
	file, err := os.Open(input)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", input, err)
	}
	defer file.Close()

	var locations []location
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := locationRE.FindStringSubmatch(scanner.Text())
		if matches == nil {
			continue
		}
		line, err := strconv.Atoi(matches[3])
		if err != nil {
			continue
		}
		locations = append(locations, location{
			Status: matches[1],
			File:   matches[2],
			Line:   line,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", input, err)
	}
	return locations, nil
}

func buildRecords(locations []location) ([]record, error) {
	byFile := map[string][]location{}
	for _, loc := range locations {
		byFile[loc.File] = append(byFile[loc.File], loc)
	}

	var records []record
	for file, fileLocations := range byFile {
		lines, err := readLines(file)
		if err != nil {
			return nil, err
		}
		for _, loc := range fileLocations {
			if loc.Line <= 0 || loc.Line > len(lines) {
				continue
			}
			rel := relativePath(file)
			records = append(records, record{
				Epic:    epicName(rel),
				Feature: filepath.Base(file),
				Status:  loc.Status,
				Text:    strings.TrimSpace(lines[loc.Line-1]),
				Sample:  fmt.Sprintf("%s:%d", rel, loc.Line),
			})
		}
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Epic != records[j].Epic {
			return records[i].Epic < records[j].Epic
		}
		if records[i].Feature != records[j].Feature {
			return records[i].Feature < records[j].Feature
		}
		if records[i].Status != records[j].Status {
			return records[i].Status < records[j].Status
		}
		return records[i].Text < records[j].Text
	})
	return records, nil
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open feature %s: %w", path, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan feature %s: %w", path, err)
	}
	return lines, nil
}

func relativePath(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(wd, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

func epicName(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) >= 2 && parts[0] == "features" {
		return parts[1]
	}
	return "(unknown)"
}

func printReport(input string, records []record) {
	fmt.Printf("# BDD inventory from %s\n\n", input)
	printSummary(records)
	fmt.Println()
	printTopSteps(records)
	fmt.Println()
	printDetails(records)
}

func printSummary(records []record) {
	type summary struct {
		Total     int
		Undefined int
		Pending   int
	}

	byEpic := map[string]summary{}
	for _, rec := range records {
		item := byEpic[rec.Epic]
		item.Total++
		if rec.Status == "undefined" {
			item.Undefined++
		}
		if rec.Status == "pending" {
			item.Pending++
		}
		byEpic[rec.Epic] = item
	}

	epics := sortedKeys(byEpic)
	fmt.Println("## Summary by epic")
	fmt.Println("| Epic | Total | Undefined | Pending |")
	fmt.Println("|---|---:|---:|---:|")
	for _, epic := range epics {
		item := byEpic[epic]
		fmt.Printf("| `%s` | %d | %d | %d |\n", epic, item.Total, item.Undefined, item.Pending)
	}
}

func printTopSteps(records []record) {
	type aggregate struct {
		Count  int
		Epics  map[string]struct{}
		Status string
		Text   string
		Sample string
	}

	byStep := map[string]aggregate{}
	for _, rec := range records {
		key := rec.Status + "\x00" + rec.Text
		item := byStep[key]
		if item.Epics == nil {
			item.Epics = map[string]struct{}{}
			item.Status = rec.Status
			item.Text = rec.Text
			item.Sample = rec.Sample
		}
		item.Count++
		item.Epics[rec.Epic] = struct{}{}
		byStep[key] = item
	}

	items := make([]counter, 0, len(byStep))
	for key, item := range byStep {
		items = append(items, counter{
			Key:    key,
			Count:  item.Count,
			Epics:  len(item.Epics),
			Status: item.Status,
			Text:   item.Text,
			Sample: item.Sample,
		})
	}
	sortCounters(items)
	if len(items) > 40 {
		items = items[:40]
	}

	fmt.Println("## Top repeated step texts")
	fmt.Println("| Count | Epics | Status | Step | Sample |")
	fmt.Println("|---:|---:|---|---|---|")
	for _, item := range items {
		fmt.Printf("| %d | %d | `%s` | %s | `%s` |\n", item.Count, item.Epics, item.Status, item.Text, item.Sample)
	}
}

func printDetails(records []record) {
	type featureSummary struct {
		Total     int
		Undefined int
		Pending   int
		Steps     map[string]counter
	}

	byFeature := map[string]*featureSummary{}
	for _, rec := range records {
		key := rec.Epic + "\x00" + rec.Feature
		item := byFeature[key]
		if item == nil {
			item = &featureSummary{Steps: map[string]counter{}}
			byFeature[key] = item
		}
		item.Total++
		if rec.Status == "undefined" {
			item.Undefined++
		}
		if rec.Status == "pending" {
			item.Pending++
		}
		stepKey := rec.Status + "\x00" + rec.Text
		step := item.Steps[stepKey]
		if step.Count == 0 {
			step.Key = stepKey
			step.Status = rec.Status
			step.Text = rec.Text
			step.Sample = rec.Sample
		}
		step.Count++
		item.Steps[stepKey] = step
	}

	keys := sortedKeys(byFeature)
	fmt.Println("## Details by epic and feature")
	for _, key := range keys {
		parts := strings.Split(key, "\x00")
		item := byFeature[key]
		fmt.Printf("\n### %s / %s (%d total, %d undefined, %d pending)\n", parts[0], parts[1], item.Total, item.Undefined, item.Pending)

		steps := make([]counter, 0, len(item.Steps))
		for _, step := range item.Steps {
			steps = append(steps, step)
		}
		sortCounters(steps)
		for _, step := range steps {
			fmt.Printf("- `%s` %s _(%d, sample `%s`)_\n", step.Status, step.Text, step.Count, step.Sample)
		}
	}
}

func sortedKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortCounters(items []counter) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		if items[i].Status != items[j].Status {
			return items[i].Status < items[j].Status
		}
		return items[i].Text < items[j].Text
	})
}
