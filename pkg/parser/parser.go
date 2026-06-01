package parser

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const (
	headerPrefix  = "Heading,Time,Course"
	minTakeWeight = 1
)

type parser struct {
	session Session

	eaten  float64
	eating float64

	previousTime   float64
	previousType   string
	previousWeight float64

	pauseType  string
	pauseStart float64
}

func Parse(reader io.Reader) (Session, error) {
	p := parser{
		session: Session{
			VASResults: make(map[string][]VASResult),
		},
	}

	if err := p.parse(reader); err != nil {
		return Session{}, err
	}

	return p.session, nil
}

func (p *parser) parse(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read header: %w", err)
		}

		return ErrUnrecognizedFile
	}

	if !strings.HasPrefix(scanner.Text(), headerPrefix) {
		return ErrUnrecognizedFile
	}

	line := 1
	for scanner.Scan() {
		line++
		if err := p.process(scanner.Text()); err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read data: %w", err)
	}

	return nil
}

func (p *parser) process(line string) error {
	if strings.TrimSpace(line) == "" {
		return nil
	}

	cols, err := parseColumns(line)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMalformedRow, err)
	}
	if len(cols) == 0 || cols[0] == "" {
		return nil
	}

	switch cols[0] {
	case "Scale Reading":
		if err := p.processScaleReading(cols); err != nil {
			return err
		}
	case "Vas Result":
		if err := p.processVASResult(cols); err != nil {
			return err
		}
	case "Refill Stage":
		if err := p.processRefillStage(cols); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %q", ErrUnexpectedLabel, cols[0])
	}

	if cols[0] != "Refill Stage" && p.previousType == "Refill Stage" && p.session.Start == 0 {
		lastRefill := p.session.Refills[len(p.session.Refills)-1]
		p.session.Start = lastRefill.Time
	}

	p.previousType = cols[0]

	return nil
}

func (p *parser) processScaleReading(cols []string) error {
	if err := requireColumns(cols, 5); err != nil {
		return err
	}

	timeValue, err := numberAt(cols, 1)
	if err != nil {
		return err
	}

	weight, err := numberAt(cols, 4)
	if err != nil {
		return err
	}

	if p.pauseStart > 0 {
		p.session.Pauses = append(p.session.Pauses, Pause{
			Start: p.pauseStart,
			End:   timeValue,
			Type:  p.pauseType,
		})

		p.pauseType = ""
		p.pauseStart = 0
	}

	if p.previousWeight-weight >= minTakeWeight {
		take := Take{
			Time:    timeValue,
			Weight:  weight,
			Take:    roundToTenth(weight-p.previousWeight) * -1,
			ITI:     timeValue - p.previousTime,
			Elapsed: timeValue - p.session.Start,
		}

		p.eaten += take.Take
		p.eating += take.ITI
		take.Eaten = p.eaten
		take.Eating = p.eating

		p.session.Takes = append(p.session.Takes, take)
		p.previousTime = timeValue
	}

	p.previousWeight = weight

	return nil
}

func (p *parser) processRefillStage(cols []string) error {
	if err := requireColumns(cols, 5); err != nil {
		return err
	}

	timeValue, err := numberAt(cols, 1)
	if err != nil {
		return err
	}

	weight, err := numberAt(cols, 4)
	if err != nil {
		return err
	}

	p.session.Refills = append(p.session.Refills, Refill{
		Time:         timeValue,
		Weight:       weight,
		RefillWeight: weight - p.previousWeight,
	})

	if p.pauseStart == 0 {
		p.pauseStart = p.previousTime
		p.pauseType = "Refill"
	}

	p.previousTime = timeValue

	return nil
}

func (p *parser) processVASResult(cols []string) error {
	if err := requireColumns(cols, 6); err != nil {
		return err
	}

	timeValue, err := numberAt(cols, 1)
	if err != nil {
		return err
	}

	value, err := numberAt(cols, 4)
	if err != nil {
		return err
	}

	question := cols[5]
	if _, ok := p.session.VASResults[question]; !ok {
		p.session.VASOrder = append(p.session.VASOrder, question)
	}

	p.session.VASResults[question] = append(p.session.VASResults[question], VASResult{
		Time:  timeValue,
		Value: value,
		Eaten: p.eaten,
	})

	if p.pauseStart == 0 {
		p.pauseStart = p.previousTime
		p.pauseType = "VAS"
	}

	p.previousTime = timeValue

	return nil
}

func parseColumns(line string) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(line))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	return reader.Read()
}

func requireColumns(cols []string, count int) error {
	if len(cols) < count {
		return fmt.Errorf("%w: expected at least %d columns, got %d", ErrMalformedRow, count, len(cols))
	}

	return nil
}

func numberAt(cols []string, index int) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(cols[index]), 64)
	if err != nil {
		return 0, fmt.Errorf("%w: column %d is not numeric", ErrMalformedRow, index+1)
	}

	return value, nil
}

func roundToTenth(value float64) float64 {
	return math.Round(value*10) / 10
}
