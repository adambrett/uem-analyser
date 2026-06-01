package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adambrett/uem-analyser/pkg/parser"
)

func TestParse_ReturnsSessionData(t *testing.T) {
	// Given
	input := strings.NewReader(sampleData())

	// When
	session, err := parser.Parse(input)

	// Then
	require.NoError(t, err)
	require.Len(t, session.Refills, 2)
	require.Len(t, session.Takes, 4)
	require.Len(t, session.Pauses, 3)
	require.Len(t, session.VASResults["How FULL are you?"], 3)

	assert.Equal(t, 5.0, session.Start)
	assert.Equal(t, 100.0, session.Refills[0].Weight)
	assert.Equal(t, 1.4, session.Takes[0].Take)
	assert.Equal(t, 10.0, session.Takes[0].ITI)
	assert.Equal(t, 10.0, session.Takes[3].Eaten)
	assert.Equal(t, "VAS", session.Pauses[0].Type)
	assert.Equal(t, 15.0, session.Pauses[0].Start)
	assert.Equal(t, 25.0, session.Pauses[0].End)
	assert.Equal(t, []string{"How FULL are you?"}, session.VASOrder)
}

func TestParse_GivenBadHeader_ReturnsUnrecognizedFile(t *testing.T) {
	// Given
	input := strings.NewReader("bad header\n")

	// When
	_, err := parser.Parse(input)

	// Then
	require.Error(t, err)
	assert.True(t, errors.Is(err, parser.ErrUnrecognizedFile))
}

func TestParse_GivenUnexpectedLabel_ReturnsUnexpectedLabel(t *testing.T) {
	// Given
	input := strings.NewReader("Heading,Time,Course\nSurprise,1,,,2\n")

	// When
	_, err := parser.Parse(input)

	// Then
	require.Error(t, err)
	assert.True(t, errors.Is(err, parser.ErrUnexpectedLabel))
}

func TestParse_GivenBlankRows_SkipsThem(t *testing.T) {
	// Given
	input := strings.NewReader("Heading,Time,Course\n\nRefill Stage,0,,,100,\nScale Reading,1,,,100,\n")

	// When
	session, err := parser.Parse(input)

	// Then
	require.NoError(t, err)
	assert.Len(t, session.Refills, 1)
}

func TestParse_GivenSmallWeightChange_DoesNotCreateTake(t *testing.T) {
	// Given
	input := strings.NewReader("Heading,Time,Course\nRefill Stage,0,,,100,\nScale Reading,1,,,100,\nScale Reading,2,,,99.2,\n")

	// When
	session, err := parser.Parse(input)

	// Then
	require.NoError(t, err)
	assert.Empty(t, session.Takes)
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
