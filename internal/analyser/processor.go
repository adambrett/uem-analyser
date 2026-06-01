package analyser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/adambrett/uem-analyser/internal/spreadsheet"
	"github.com/adambrett/uem-analyser/pkg/parser"
)

const (
	mimeTypeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	mimeTypeZip  = "application/zip"
)

var vasCodePattern = regexp.MustCompile(`\b[A-Z]+\b`)

type parsedFile struct {
	name    string
	session parser.Session
}

func Inspect(files []InputFile) (Inspection, error) {
	parsed, err := parseFiles(files)
	if err != nil {
		return Inspection{}, err
	}

	groups, groupWarnings := groupByParticipant(parsed)

	return Inspection{
		Questions:    detectQuestions(parsed),
		Participants: summarizeGroups(groups),
		Warnings:     groupWarnings,
	}, nil
}

func Generate(files []InputFile, selectedVAS []string) (Download, error) {
	parsed, err := parseFiles(files)
	if err != nil {
		return Download{}, err
	}

	groups, _ := groupByParticipant(parsed)
	outputs, err := writeWorkbooks(groups, selectedVAS)
	if err != nil {
		return Download{}, err
	}
	if len(outputs) == 1 {
		return Download{
			Name:     outputs[0].name,
			MIMEType: mimeTypeXLSX,
			Data:     outputs[0].data,
		}, nil
	}

	data, err := zipOutputs(outputs)
	if err != nil {
		return Download{}, err
	}

	return Download{
		Name:     "uem-analyser-results.zip",
		MIMEType: mimeTypeZip,
		Data:     data,
	}, nil
}

func parseFiles(files []InputFile) ([]parsedFile, error) {
	if len(files) == 0 {
		return nil, ErrNoFiles
	}

	parsed := make([]parsedFile, 0, len(files))
	for _, file := range files {
		if len(file.Data) > MaxFileSize {
			return nil, FileError{Name: file.Name, Err: ErrFileTooLarge}
		}

		session, err := parser.Parse(bytes.NewReader(file.Data))
		if err != nil {
			return nil, FileError{Name: file.Name, Err: err}
		}

		parsed = append(parsed, parsedFile{
			name:    filepath.Base(file.Name),
			session: session,
		})
	}

	return parsed, nil
}

func detectQuestions(files []parsedFile) []string {
	questions := make(map[string]struct{})
	for _, file := range files {
		for _, question := range file.session.VASOrder {
			code, ok := VASCode(question)
			if ok {
				questions[code] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(questions))
	for question := range questions {
		result = append(result, question)
	}
	sort.Strings(result)

	return result
}

func VASCode(question string) (string, bool) {
	match := vasCodePattern.FindString(question)
	if match == "" {
		return "", false
	}

	return match, true
}

func groupByParticipant(files []parsedFile) (map[string][]parsedFile, []Warning) {
	groups := make(map[string][]parsedFile)
	var warnings []Warning

	for _, file := range files {
		participant, ok := participantName(file.name)
		if !ok {
			warnings = append(warnings, Warning{
				File:    file.name,
				Message: "Filename does not contain \"session\"; using the filename as the participant name.",
			})
		}

		groups[participant] = append(groups[participant], file)
	}

	for participant := range groups {
		sort.Slice(groups[participant], func(i, j int) bool {
			return groups[participant][i].name < groups[participant][j].name
		})
	}

	return groups, warnings
}

func summarizeGroups(groups map[string][]parsedFile) []ParticipantSummary {
	names := sortedParticipantNames(groups)
	summaries := make([]ParticipantSummary, 0, len(names))

	for _, name := range names {
		files := make([]string, 0, len(groups[name]))
		for _, file := range groups[name] {
			files = append(files, file.name)
		}

		summaries = append(summaries, ParticipantSummary{
			Name:  name,
			Files: files,
		})
	}

	return summaries
}

func participantName(filename string) (string, bool) {
	base := filepath.Base(filename)
	index := strings.Index(strings.ToLower(base), "session")
	if index > 0 {
		name := base[:index]
		if strings.TrimSpace(name) != "" {
			return name, true
		}
	}

	extension := filepath.Ext(base)
	name := strings.TrimSuffix(base, extension)
	if name == "" {
		name = "results"
	}

	return name, false
}

func sortedParticipantNames(groups map[string][]parsedFile) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

type workbookOutput struct {
	name string
	data []byte
}

func writeWorkbooks(groups map[string][]parsedFile, selectedVAS []string) ([]workbookOutput, error) {
	names := sortedParticipantNames(groups)
	outputs := make([]workbookOutput, 0, len(names))

	for _, name := range names {
		sessions := make([]spreadsheet.SessionFile, 0, len(groups[name]))
		for _, file := range groups[name] {
			sessions = append(sessions, spreadsheet.SessionFile{
				Name:    file.name,
				Session: file.session,
			})
		}

		data, err := spreadsheet.WriteWorkbook(spreadsheet.Workbook{
			Participant: name,
			Sessions:    sessions,
		}, selectedVAS)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}

		outputs = append(outputs, workbookOutput{
			name: fmt.Sprintf("%s Results.xlsx", name),
			data: data,
		})
	}

	return outputs, nil
}

func zipOutputs(outputs []workbookOutput) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, output := range outputs {
		fileWriter, err := writer.Create(output.name)
		if err != nil {
			return nil, fmt.Errorf("create zip entry: %w", err)
		}
		if _, err := fileWriter.Write(output.data); err != nil {
			return nil, fmt.Errorf("write zip entry: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buffer.Bytes(), nil
}
