package spreadsheet_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/adambrett/uem-analyser/internal/spreadsheet"
	"github.com/adambrett/uem-analyser/pkg/parser"
)

func TestWriteWorkbook_WritesLegacySections(t *testing.T) {
	// Given
	session, err := parser.Parse(strings.NewReader(sampleData()))
	require.NoError(t, err)

	// When
	data, err := spreadsheet.WriteWorkbook(spreadsheet.Workbook{
		Participant: "Ada ",
		Sessions: []spreadsheet.SessionFile{
			{Name: "Ada session 1.txt", Session: session},
		},
	}, []string{"FULL"})

	// Then
	require.NoError(t, err)

	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer workbook.Close()

	assert.Equal(t, "session 1.txt", workbook.GetSheetName(0))
	assertCell(t, workbook, "session 1.txt", "A1", "Take")
	assertCell(t, workbook, "session 1.txt", "G2", "1.4")
	assertCell(t, workbook, "session 1.txt", "A8", "File Time")
	assertCell(t, workbook, "session 1.txt", "A13", "Start (s)")
	assertCell(t, workbook, "session 1.txt", "A19", "Minute")
	assertCell(t, workbook, "session 1.txt", "A25", "Quartile")
	assertCell(t, workbook, "session 1.txt", "A32", "How FULL are you?")
}

func TestWriteWorkbook_FiltersSelectedVASQuestions(t *testing.T) {
	// Given
	session, err := parser.Parse(strings.NewReader(sampleDataWithTwoVAS()))
	require.NoError(t, err)

	// When
	data, err := spreadsheet.WriteWorkbook(spreadsheet.Workbook{
		Participant: "Ada ",
		Sessions: []spreadsheet.SessionFile{
			{Name: "Ada session 1.txt", Session: session},
		},
	}, []string{"FULL"})

	// Then
	require.NoError(t, err)

	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer workbook.Close()

	text := sheetText(t, workbook, "session 1.txt")
	assert.Contains(t, text, "How FULL are you?")
	assert.NotContains(t, text, "How HUNGRY are you?")
}

func TestWriteWorkbook_GivenNoSessions_ReturnsError(t *testing.T) {
	// When
	_, err := spreadsheet.WriteWorkbook(spreadsheet.Workbook{Participant: "Ada"}, []string{"FULL"})

	// Then
	require.ErrorIs(t, err, spreadsheet.ErrNoSessions)
}

func TestWriteWorkbook_GivenNoTakes_ReturnsError(t *testing.T) {
	// Given
	session, err := parser.Parse(strings.NewReader("Heading,Time,Course\nRefill Stage,0,,,100,\nScale Reading,1,,,100,\n"))
	require.NoError(t, err)

	// When
	_, err = spreadsheet.WriteWorkbook(spreadsheet.Workbook{
		Participant: "Ada",
		Sessions: []spreadsheet.SessionFile{
			{Name: "Ada session 1.txt", Session: session},
		},
	}, []string{"FULL"})

	// Then
	require.ErrorIs(t, err, spreadsheet.ErrNoTakes)
}

func assertCell(t *testing.T, workbook *excelize.File, sheet, cell, expected string) {
	t.Helper()

	value, err := workbook.GetCellValue(sheet, cell)
	require.NoError(t, err)
	assert.Equal(t, expected, value)
}

func sheetText(t *testing.T, workbook *excelize.File, sheet string) string {
	t.Helper()

	rows, err := workbook.GetRows(sheet)
	require.NoError(t, err)

	var text strings.Builder
	for _, row := range rows {
		for _, value := range row {
			text.WriteString(value)
			text.WriteByte('\n')
		}
	}

	return text.String()
}

func sampleData() string {
	return strings.Join([]string{
		"Heading,Time,Course",
		"Refill Stage,5,,,100,",
		"Scale Reading,10,,,100,",
		"Scale Reading,15,,,98.6,",
		"Vas Result,20,,,40,How FULL are you?",
		"Scale Reading,25,,,98.6,",
		"Scale Reading,70,,,96.0,",
		"Vas Result,100,,,50,How FULL are you?",
		"Scale Reading,120,,,94.0,",
		"Refill Stage,130,,,120,",
		"Scale Reading,135,,,120,",
		"Scale Reading,180,,,116,",
		"Vas Result,200,,,70,How FULL are you?",
	}, "\n")
}

func sampleDataWithTwoVAS() string {
	return strings.Join([]string{
		"Heading,Time,Course",
		"Refill Stage,0,,,100,",
		"Scale Reading,1,,,100,",
		"Scale Reading,5,,,98,",
		"Vas Result,10,,,40,How FULL are you?",
		"Vas Result,15,,,20,How HUNGRY are you?",
		"Scale Reading,70,,,96,",
		"Vas Result,80,,,50,How FULL are you?",
		"Vas Result,85,,,30,How HUNGRY are you?",
	}, "\n")
}
