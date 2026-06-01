package analyser

const MaxFileSize = 5 * 1024 * 1024

type InputFile struct {
	Name string
	Data []byte
}

type Inspection struct {
	Questions    []string             `json:"questions"`
	Participants []ParticipantSummary `json:"participants"`
	Warnings     []Warning            `json:"warnings"`
}

type ParticipantSummary struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

type Warning struct {
	File    string `json:"file,omitempty"`
	Message string `json:"message"`
}

type Download struct {
	Name     string
	MIMEType string
	Data     []byte
}
