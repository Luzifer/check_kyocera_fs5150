package main

import (
	"flag"
	"fmt"
	"math"
	"strings"

	"github.com/alouca/gosnmp"
	"github.com/nightlyone/go-nagios/nagios"
)

type tonerlevels map[string]int

func main() {

	critical := flag.Int("c", 5, "Threshold of toner left for critical state")
	warning := flag.Int("w", 10, "Threshold of toner left for warning state")
	host := flag.String("H", "localhost", "Hostname of Kyocera printer")
	community := flag.String("C", "public", "SNMP community to use for data queries")
	flag.Parse()

	levels, err := getTonerLevel(*host, *community)
	if err != nil {
		nagios.Exit(nagios.CRITICAL, err.Error())
	}

	messages := []string{}
	state := NewReturnstate(*critical, *warning)
	for k, v := range levels {
		messages = append(messages, getMessageForTonerLevel(k, v))
		state.update(v)
	}

	nagios.Exit(state.getStatus(), strings.Join(messages, ", "))

}

type returnstate struct {
	nagios.Status
	critical, warning int
}

func NewReturnstate(critical, warning int) *returnstate {
	return &returnstate{
		Status:   nagios.OK,
		critical: critical,
		warning:  warning,
	}
}

func (state *returnstate) update(level int) {
	newState := nagios.OK
	switch {
	case level <= state.critical:
		newState = nagios.CRITICAL
	case level <= state.warning:
		newState = nagios.WARNING
	}

	if newState > state.Status {
		state.Status = newState
	}
}

func (state *returnstate) getStatus() nagios.Status {
	return state.Status
}

func getMessageForTonerLevel(color string, level int) string {
	switch {
	case level >= 0:
		return fmt.Sprintf("%s is at %d%%", color, level)
	case level == -2:
		return fmt.Sprintf("%s is not present", color)
	default:
		return fmt.Sprintf("%s has an unknown state")
	}
}

func getTonerLevel(hostname, community string) (tonerlevels, error) {
	levels := make(tonerlevels)
	snmp, err := gosnmp.NewGoSNMP(hostname, community, gosnmp.Version2c, 5)
	if err != nil {
		return nil, err
	}

	for i := 1; i <= 4; i++ {
		colorIdent := fmt.Sprintf("1.3.6.1.2.1.43.12.1.1.4.1.%d", i)
		maxLevelIdent := fmt.Sprintf("1.3.6.1.2.1.43.11.1.1.8.1.%d", i)
		currentLevelIdent := fmt.Sprintf("1.3.6.1.2.1.43.11.1.1.9.1.%d", i)

		resp, err := snmp.Get(colorIdent)
		if err != nil {
			return nil, err
		}
		color := string(resp.Variables[0].Value.([]byte))

		resp, err = snmp.Get(maxLevelIdent)
		if err != nil {
			return nil, err
		}
		maxLevel := resp.Variables[0].Value.(int)

		resp, err = snmp.Get(currentLevelIdent)
		if err != nil {
			return nil, err
		}
		currentLevel := resp.Variables[0].Value.(int)

		if maxLevel > 0 {
			levels[color] = int(math.Floor(float64(currentLevel) / float64(maxLevel) * 100.0))
		} else {
			levels[color] = maxLevel
		}
	}

	return levels, nil
}
