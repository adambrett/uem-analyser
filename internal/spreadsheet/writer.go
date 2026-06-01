package spreadsheet

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/adambrett/uem-analyser/pkg/parser"
)

var (
	ErrNoSessions = errors.New("workbook has no sessions")
	ErrNoTakes    = errors.New("session has no takes")

	vasCodePattern = regexp.MustCompile(`\b[A-Z]+\b`)
)

type annotatedTake struct {
	Base     parser.Take
	Minute   int
	Quartile int
}

func WriteWorkbook(workbook Workbook, selectedVAS []string) ([]byte, error) {
	if len(workbook.Sessions) == 0 {
		return nil, ErrNoSessions
	}

	file := excelize.NewFile()
	defer func() {
		_ = file.Close()
	}()

	if err := file.SetDocProps(&excelize.DocProperties{
		Creator:        "Sarah Santos-Murphy",
		LastModifiedBy: "Sarah Santos-Murphy",
		Title:          workbook.Participant + " Results",
		Description:    "MS3 output, generated using http://sarah.santos-murphy.com/ms3.",
	}); err != nil {
		return nil, fmt.Errorf("set workbook properties: %w", err)
	}

	sheetNames := make(map[string]struct{})
	for index, session := range workbook.Sessions {
		sheetName := uniqueSheetName(sessionSheetName(session.Name), sheetNames)
		if index == 0 {
			if err := file.SetSheetName("Sheet1", sheetName); err != nil {
				return nil, fmt.Errorf("rename first sheet: %w", err)
			}
		} else if _, err := file.NewSheet(sheetName); err != nil {
			return nil, fmt.Errorf("create sheet: %w", err)
		}

		if err := writeSession(file, sheetName, session.Session, selectedVAS); err != nil {
			return nil, fmt.Errorf("%s: %w", session.Name, err)
		}
	}

	file.SetActiveSheet(0)

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write workbook: %w", err)
	}

	return buffer.Bytes(), nil
}

func writeSession(file *excelize.File, sheet string, session parser.Session, selectedVAS []string) error {
	takes, err := annotateTakes(session.Takes)
	if err != nil {
		return err
	}

	row := 1
	writer := cellWriter{file: file, sheet: sheet}

	writer.addTakes(&row, takes)
	row += 2
	writer.addRefills(&row, session.Refills)
	row += 2
	writer.addPauses(&row, session.Pauses)
	row += 2
	writer.addMinuteStats(&row, takes)
	row += 2
	writer.addQuartileStats(&row, takes)
	row += 2
	writer.addVAS(&row, session, selectedVAS)
	row += 2
	writer.addVASQuartile(&row, session, selectedVAS)
	row += 2
	writer.addVASMinutes(&row, session, selectedVAS)

	return writer.err
}

func annotateTakes(takes []parser.Take) ([]annotatedTake, error) {
	if len(takes) == 0 {
		return nil, ErrNoTakes
	}

	last := takes[len(takes)-1]
	quarter := last.Eaten / 4
	if quarter <= 0 {
		return nil, ErrNoTakes
	}

	annotated := make([]annotatedTake, 0, len(takes))
	for _, take := range takes {
		quartile := int(math.Floor(take.Eaten/quarter)) + 1
		if quartile == 5 {
			quartile = 4
		}

		annotated = append(annotated, annotatedTake{
			Base:     take,
			Minute:   int(math.Floor(take.Eating/60)) + 1,
			Quartile: quartile,
		})
	}

	return annotated, nil
}

type cellWriter struct {
	file  *excelize.File
	sheet string
	err   error
}

func (w *cellWriter) set(column string, row int, value any) {
	if w.err != nil {
		return
	}

	cell := fmt.Sprintf("%s%d", column, row)
	w.err = w.file.SetCellValue(w.sheet, cell, value)
}

func (w *cellWriter) addTakes(row *int, takes []annotatedTake) {
	headers := []any{
		"Take", "File Time (s)", "Inter-take Interval (s)", "Eating (s)",
		"Elapsed (s)", "File Weight (g)", "Take (g)", "Takes/minute",
		"Average Take (g)", "Eaten (g)", "Eaten Weight/minute", "Quartile", "Minutes",
	}
	w.writeRow(row, headers)

	for index, take := range takes {
		values := []any{
			index + 1,
			format1(take.Base.Time),
			format1(take.Base.ITI),
			format1(take.Base.Eating),
			format1(take.Base.Elapsed),
			take.Base.Weight,
			take.Base.Take,
			format1(safeDiv(float64(index+1), take.Base.Elapsed/60)),
			format1(safeDiv(take.Base.Eaten, float64(index+1))),
			take.Base.Eaten,
			format1(safeDiv(take.Base.Eaten, take.Base.Elapsed/60)),
			format0(float64(take.Quartile)),
			format1(float64(take.Minute)),
		}
		w.writeRow(row, values)
	}
}

func (w *cellWriter) addRefills(row *int, refills []parser.Refill) {
	w.writeRow(row, []any{"File Time", "File Weight", "Refill", "Dish"})

	total := 0.0
	for _, refill := range refills {
		total += refill.Weight
		w.writeRow(row, []any{
			format1(refill.Time),
			format1(refill.Weight),
			format1(refill.RefillWeight),
			format1(total),
		})
	}
}

func (w *cellWriter) addPauses(row *int, pauses []parser.Pause) {
	w.writeRow(row, []any{"Start (s)", "End (s)", "Pause (s)", "Type", "Pause Total (s)"})

	total := 0.0
	for _, pause := range pauses {
		duration := pause.End - pause.Start
		total += duration

		w.writeRow(row, []any{
			format1(pause.Start),
			format1(pause.End),
			format1(duration),
			pause.Type,
			format1(total),
		})
	}
}

func (w *cellWriter) addMinuteStats(row *int, takes []annotatedTake) {
	w.writeRow(row, []any{"Minute", "Takes", "Average Take (g)", "Take (g)", "Eaten (g)"})

	eaten := 0.0
	previousEaten := 0.0
	previousTakes := 0
	previousMinute := 1
	minute := 1

	for index, take := range takes {
		minute = take.Minute
		takeCount := index - previousTakes

		if minute != previousMinute {
			if index == 0 {
				previousMinute = minute
				continue
			}

			previousTake := takes[index-1]
			takeValue := previousTake.Base.Eaten - previousEaten
			w.writeRow(row, []any{
				previousMinute,
				takeCount,
				format1(safeDiv(takeValue, float64(takeCount))),
				format1(takeValue),
				format1(eaten),
			})

			if minute > previousMinute+1 {
				for i := previousMinute + 1; i < minute-1; i++ {
					w.writeRow(row, []any{i, 0, 0, 0, format1(eaten)})
				}
			}

			previousMinute = minute
			previousTakes = index
			previousEaten = previousTake.Base.Eaten
		}

		eaten += take.Base.Take
	}

	takeCount := len(takes) - previousTakes
	last := takes[len(takes)-1]
	takeValue := last.Base.Eaten - previousEaten
	w.writeRow(row, []any{
		minute,
		takeCount,
		format1(safeDiv(takeValue, float64(takeCount))),
		format1(takeValue),
		format1(eaten),
	})
}

func (w *cellWriter) addQuartileStats(row *int, takes []annotatedTake) {
	w.writeRow(row, []any{
		"Quartile", "Elapsed (s)", "Duration (s)", "Eating (s)", "Takes",
		"Average Takes (g)", "Take (g)", "Eaten (g)", "Takes/minute",
	})

	previousEaten := 0.0
	previousElapsed := 0.0
	previousTakes := 0
	previousQuartile := 1

	for index, take := range takes {
		quartile := take.Quartile
		if quartile != previousQuartile {
			if index == 0 {
				previousQuartile = quartile
				continue
			}

			previousTake := takes[index-1]
			takeValue := previousTake.Base.Eaten - previousEaten
			takeCount := index - previousTakes
			duration := previousTake.Base.Elapsed - previousElapsed

			w.writeRow(row, []any{
				quartile - 1,
				format1(previousTake.Base.Elapsed),
				format1(duration),
				format1(previousTake.Base.Eating),
				takeCount,
				format1(safeDiv(takeValue, float64(takeCount))),
				takeValue,
				format1(previousTake.Base.Eaten),
				format2(safeDiv(takeValue, duration/60)),
			})

			previousQuartile = take.Quartile
			previousTakes = index
			previousEaten = previousTake.Base.Eaten
			previousElapsed = previousTake.Base.Elapsed
		}
	}

	takeCount := len(takes) - previousTakes
	last := takes[len(takes)-1]
	takeValue := last.Base.Eaten - previousEaten
	duration := last.Base.Elapsed - previousElapsed

	w.writeRow(row, []any{
		4,
		format1(last.Base.Elapsed),
		format1(duration),
		format1(last.Base.Eating),
		takeCount,
		format1(safeDiv(takeValue, float64(takeCount))),
		takeValue,
		format1(last.Base.Eaten),
		format2(safeDiv(takeValue, duration/60)),
	})
}

func (w *cellWriter) addVAS(row *int, session parser.Session, selectedVAS []string) {
	first := true
	for _, question := range session.VASOrder {
		code, ok := vasCode(question)
		if !ok || !isSelected(code, selectedVAS) {
			continue
		}

		if !first {
			*row += 2
		}

		w.set("A", *row, question)
		*row++
		w.writeRow(row, []any{"File Time (s)", "Eaten (g)", code})

		for _, result := range session.VASResults[question] {
			w.writeRow(row, []any{
				format1(result.Time),
				result.Eaten,
				result.Value,
			})
		}

		first = false
	}
}

func (w *cellWriter) addVASQuartile(row *int, session parser.Session, selectedVAS []string) {
	first := true
	for _, question := range session.VASOrder {
		code, ok := vasCode(question)
		if !ok || !isSelected(code, selectedVAS) {
			continue
		}

		results := session.VASResults[question]
		if len(results) == 0 {
			continue
		}

		if !first {
			*row += 2
		}

		w.set("A", *row, question)
		*row++
		w.writeRow(row, []any{"Quartile (g)", "Previous Eaten (g)", "Eaten (g)", "Previous VAS", code})
		w.writeRow(row, []any{0, 0, 0, 0, "(" + format1(results[0].Value) + ")"})

		last := results[len(results)-1]
		quarter := last.Eaten / 4
		if quarter <= 0 {
			first = false
			continue
		}

		quartileNumber := 1
		quartileGrams := quarter
		previousVAS := results[0].Value
		previousEaten := results[0].Eaten

		for _, result := range results[1:] {
			for result.Eaten >= quartileGrams {
				interpolatedVAS := interpolate(previousEaten, result.Eaten, previousVAS, result.Value, quartileGrams)
				w.writeRow(row, []any{
					quartileGrams,
					previousEaten,
					result.Eaten,
					previousVAS,
					format1(interpolatedVAS),
				})

				quartileNumber++
				quartileGrams = quarter * float64(quartileNumber)
			}

			previousEaten = result.Eaten
			previousVAS = result.Value
		}

		first = false
	}
}

func (w *cellWriter) addVASMinutes(row *int, session parser.Session, selectedVAS []string) {
	first := true
	for _, question := range session.VASOrder {
		code, ok := vasCode(question)
		if !ok || !isSelected(code, selectedVAS) {
			continue
		}

		results := session.VASResults[question]
		if len(results) == 0 {
			continue
		}

		if !first {
			*row += 2
		}

		w.set("A", *row, question)
		*row++
		w.writeRow(row, []any{"Minute", "Previous VAS", code})
		w.writeRow(row, []any{0, 0, "(" + format1(results[0].Value) + ")"})

		elapsed := 60.0
		previousTime := 0.0
		previousVAS := 0.0

		for _, result := range results {
			for result.Time > elapsed {
				interpolatedVAS := interpolate(previousTime, result.Time, previousVAS, result.Value, elapsed)
				w.writeRow(row, []any{
					elapsed / 60,
					previousVAS,
					format1(interpolatedVAS),
				})

				elapsed += 60
			}

			previousTime = result.Time
			previousVAS = result.Value
		}

		first = false
	}
}

func (w *cellWriter) writeRow(row *int, values []any) {
	for index, value := range values {
		w.set(columnName(index), *row, value)
	}
	*row++
}

func columnName(index int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	name := ""
	for index >= 0 {
		remainder := index % len(alphabet)
		name = alphabet[remainder:remainder+1] + name
		index = index/len(alphabet) - 1
	}
	return name
}

func format0(value float64) string {
	return fmt.Sprintf("%.0f", value)
}

func format1(value float64) string {
	return fmt.Sprintf("%.1f", value)
}

func format2(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func safeDiv(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func interpolate(xLow, xHigh, yLow, yHigh, xTarget float64) float64 {
	xDiff := xHigh - xLow
	if xDiff == 0 {
		return yLow
	}

	yDiff := yHigh - yLow
	xIncrement := xTarget - xLow

	return yLow + ((yDiff / xDiff) * xIncrement)
}

func vasCode(question string) (string, bool) {
	match := vasCodePattern.FindString(question)
	if match == "" {
		return "", false
	}

	return match, true
}

func isSelected(code string, selectedVAS []string) bool {
	for _, selected := range selectedVAS {
		if selected == code {
			return true
		}
	}

	return false
}

func sessionSheetName(filename string) string {
	base := filepath.Base(filename)
	index := strings.Index(strings.ToLower(base), "session")
	if index >= 0 {
		return base[index:]
	}

	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "session"
	}

	return name
}

func uniqueSheetName(name string, used map[string]struct{}) string {
	clean := sanitizeSheetName(name)
	if _, ok := used[clean]; !ok {
		used[clean] = struct{}{}
		return clean
	}

	for suffix := 2; ; suffix++ {
		suffixText := fmt.Sprintf(" %d", suffix)
		base := clean
		if len(base)+len(suffixText) > 31 {
			base = base[:31-len(suffixText)]
		}

		candidate := base + suffixText
		if _, ok := used[candidate]; !ok {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func sanitizeSheetName(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		clean = "session"
	}

	replacer := strings.NewReplacer("\\", "_", "/", "_", "?", "_", "*", "_", "[", "_", "]", "_", ":", "_")
	clean = replacer.Replace(clean)
	if len(clean) > 31 {
		clean = clean[:31]
	}

	return clean
}
