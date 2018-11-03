package main

import (
	"strings"
)

var longCommand = []string{"SLP", "NSA", "NSE"}

func parseData(data string) (endpoint, payload string) {
	if strings.HasPrefix(data, "Z2") {
		endpoint = "Z2"
		data = data[2:]
	}

	for _, lcmd := range longCommand {
		if strings.HasPrefix(data, lcmd) {
			endpoint += lcmd
			payload = data[3:]
			return
		}
	}

	if strings.HasPrefix(data, "CV") {
		e, d := parseCVCmd(data)
		endpoint += e
		payload = d
		return
	}

	if strings.HasPrefix(data, "MV") {
		e, d := parseMVCmd(data)
		endpoint += e
		payload = d
		return
	}

	if strings.HasPrefix(data, "PS") {
		e, d := parsePSCmd(data)
		endpoint += e
		payload = d
		return
	}

	endpoint += data[:2]
	payload = data[2:]

	return
}

func parseCVCmd(data string) (string, string) {
	parts := strings.Fields(data)
	return parts[0], strings.Join(parts[1:], " ")
}

func parsePSCmd(data string) (string, string) {
	if strings.HasPrefix(data, "PSMODE") || strings.HasPrefix(data, "PSMULTEQ") {
		parts := strings.Split(data, ":")
		typ := parts[0]
		data = parts[1]
		return typ, data
	}

	parts := strings.Fields(data)
	typ := parts[0]
	data = strings.Join(parts[1:], " ")
	return typ, data
}

func parseMVCmd(data string) (string, string) {
	if strings.HasPrefix(data, "MVMAX") {
		parts := strings.Fields(data)
		typ := parts[0]
		data = strings.Join(parts[1:], " ")
		return typ, data
	}
	return data[:2], data[2:]
}
