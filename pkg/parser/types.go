package parser

type Session struct {
	Takes      []Take
	Refills    []Refill
	Pauses     []Pause
	VASResults map[string][]VASResult
	VASOrder   []string
	Start      float64
}

type Take struct {
	Time    float64
	Weight  float64
	Take    float64
	ITI     float64
	Elapsed float64
	Eaten   float64
	Eating  float64
}

type Refill struct {
	Time         float64
	Weight       float64
	RefillWeight float64
}

type Pause struct {
	Start float64
	End   float64
	Type  string
}

type VASResult struct {
	Time  float64
	Value float64
	Eaten float64
}
