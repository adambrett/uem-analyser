package analyser_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/adambrett/uem-analyser/internal/analyser"
)

func TestInspect_ReturnsQuestionsAndParticipants(t *testing.T) {
	// Given
	files := []analyser.InputFile{
		{Name: "Ada session 1.txt", Data: []byte(sampleData("How FULL are you?"))},
		{Name: "Ada session 2.txt", Data: []byte(sampleData("How HUNGRY are you?"))},
	}

	// When
	inspection, err := analyser.Inspect(files)

	// Then
	require.NoError(t, err)
	assert.Equal(t, []string{"FULL", "HUNGRY"}, inspection.Questions)
	require.Len(t, inspection.Participants, 1)
	assert.Equal(t, "Ada ", inspection.Participants[0].Name)
	assert.Equal(t, []string{"Ada session 1.txt", "Ada session 2.txt"}, inspection.Participants[0].Files)
}

func TestInspect_GivenNoFiles_ReturnsNoFiles(t *testing.T) {
	// When
	_, err := analyser.Inspect(nil)

	// Then
	require.Error(t, err)
	assert.True(t, errors.Is(err, analyser.ErrNoFiles))
}

func TestInspect_GivenFilenameWithoutSession_ReturnsWarning(t *testing.T) {
	// Given
	files := []analyser.InputFile{
		{Name: "Ada.txt", Data: []byte(sampleData("How FULL are you?"))},
	}

	// When
	inspection, err := analyser.Inspect(files)

	// Then
	require.NoError(t, err)
	require.Len(t, inspection.Warnings, 1)
	assert.Equal(t, "Ada.txt", inspection.Warnings[0].File)
	assert.Equal(t, "Ada", inspection.Participants[0].Name)
}

func TestInspect_GivenTooLargeFile_ReturnsFileError(t *testing.T) {
	// Given
	files := []analyser.InputFile{
		{Name: "Ada session 1.txt", Data: bytes.Repeat([]byte("x"), analyser.MaxFileSize+1)},
	}

	// When
	_, err := analyser.Inspect(files)

	// Then
	require.Error(t, err)
	assert.True(t, errors.Is(err, analyser.ErrFileTooLarge))
}

func TestInspect_GivenBadFile_ReturnsNamedParseError(t *testing.T) {
	// Given
	files := []analyser.InputFile{
		{Name: "Ada session 1.txt", Data: []byte("not a UEM file\n")},
	}

	// When
	_, err := analyser.Inspect(files)

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ada session 1.txt")
}

func TestGenerate_GivenOneParticipant_ReturnsXLSX(t *testing.T) {
	// Given
	files := []analyser.InputFile{
		{Name: "Ada session 1.txt", Data: []byte(sampleData("How FULL are you?"))},
	}

	// When
	download, err := analyser.Generate(files, []string{"FULL"})

	// Then
	require.NoError(t, err)
	assert.Equal(t, "Ada  Results.xlsx", download.Name)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", download.MIMEType)

	workbook, err := excelize.OpenReader(bytes.NewReader(download.Data))
	require.NoError(t, err)
	defer workbook.Close()

	assert.Equal(t, "session 1.txt", workbook.GetSheetName(0))
}

func TestGenerate_GivenMultipleParticipants_ReturnsZip(t *testing.T) {
	// Given
	files := []analyser.InputFile{
		{Name: "Ada session 1.txt", Data: []byte(sampleData("How FULL are you?"))},
		{Name: "Grace session 1.txt", Data: []byte(sampleData("How FULL are you?"))},
	}

	// When
	download, err := analyser.Generate(files, []string{"FULL"})

	// Then
	require.NoError(t, err)
	assert.Equal(t, "uem-analyser-results.zip", download.Name)
	assert.Equal(t, "application/zip", download.MIMEType)

	reader, err := zip.NewReader(bytes.NewReader(download.Data), int64(len(download.Data)))
	require.NoError(t, err)
	require.Len(t, reader.File, 2)

	var names []string
	for _, file := range reader.File {
		names = append(names, file.Name)

		entry, err := file.Open()
		require.NoError(t, err)
		_, err = io.ReadAll(entry)
		require.NoError(t, err)
		require.NoError(t, entry.Close())
	}

	assert.Equal(t, []string{"Ada  Results.xlsx", "Grace  Results.xlsx"}, names)
}

func TestVASCode_ReturnsFirstUppercaseToken(t *testing.T) {
	// Given
	question := "How FULL are you?"

	// When
	code, ok := analyser.VASCode(question)

	// Then
	assert.True(t, ok)
	assert.Equal(t, "FULL", code)
}

func TestFixtures_ValidFilesInspectAndGenerate(t *testing.T) {
	// Given
	files := inputFilesFromDir(t, "../../fixtures/valid")

	// When
	inspection, err := analyser.Inspect(files)

	// Then
	require.NoError(t, err)
	assert.Equal(t, []string{"FULL", "HUNGRY", "THIRSTY"}, inspection.Questions)
	require.Len(t, inspection.Warnings, 1)
	assert.Equal(t, "StandaloneExample.txt", inspection.Warnings[0].File)

	download, err := analyser.Generate(files, inspection.Questions)
	require.NoError(t, err)
	assert.Equal(t, "uem-analyser-results.zip", download.Name)
	assert.Equal(t, "application/zip", download.MIMEType)
}

func TestFixtures_InvalidFilesReturnErrors(t *testing.T) {
	// Given
	files := inputFilesFromDir(t, "../../fixtures/invalid")

	// Then
	for _, file := range files {
		t.Run(file.Name, func(t *testing.T) {
			// When
			_, err := analyser.Inspect([]analyser.InputFile{file})

			// Then
			require.Error(t, err)
			assert.Contains(t, err.Error(), file.Name)
		})
	}
}

func inputFilesFromDir(t *testing.T, dir string) []analyser.InputFile {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	files := make([]analyser.InputFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		files = append(files, analyser.InputFile{
			Name: entry.Name(),
			Data: data,
		})
	}

	return files
}

func sampleData(question string) string {
	return strings.Join([]string{
		"Heading,Time,Course",
		"Refill Stage,5,,,100,",
		"Scale Reading,10,,,100,",
		"Scale Reading,15,,,98.6,",
		"Vas Result,20,,,40," + question,
		"Scale Reading,25,,,98.6,",
		"Scale Reading,70,,,96.0,",
		"Vas Result,100,,,50," + question,
		"Scale Reading,120,,,94.0,",
		"Refill Stage,130,,,120,",
		"Scale Reading,135,,,120,",
		"Scale Reading,180,,,116,",
		"Vas Result,200,,,70," + question,
	}, "\n")
}
