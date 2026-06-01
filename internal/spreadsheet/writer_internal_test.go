package spreadsheet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adambrett/uem-analyser/pkg/parser"
)

func TestAnnotateTakes_AssignsMinutesAndQuartiles(t *testing.T) {
	// Given
	takes := []parser.Take{
		{Eating: 10, Eaten: 1, Take: 1},
		{Eating: 61, Eaten: 3, Take: 2},
		{Eating: 120, Eaten: 5, Take: 2},
		{Eating: 181, Eaten: 8, Take: 3},
	}

	// When
	annotated, err := annotateTakes(takes)

	// Then
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3, 4}, takeQuartiles(annotated))
	assert.Equal(t, []int{1, 2, 3, 4}, takeMinutes(annotated))
}

func TestAnnotateTakes_GivenEmptyInput_ReturnsNoTakes(t *testing.T) {
	// When
	_, err := annotateTakes(nil)

	// Then
	require.ErrorIs(t, err, ErrNoTakes)
}

func TestInterpolate_ReturnsLinearValue(t *testing.T) {
	// When
	value := interpolate(0, 10, 20, 40, 5)

	// Then
	assert.Equal(t, 30.0, value)
}

func TestInterpolate_GivenNoRange_ReturnsLowerValue(t *testing.T) {
	// When
	value := interpolate(10, 10, 20, 40, 10)

	// Then
	assert.Equal(t, 20.0, value)
}

func takeQuartiles(takes []annotatedTake) []int {
	quartiles := make([]int, 0, len(takes))
	for _, take := range takes {
		quartiles = append(quartiles, take.Quartile)
	}

	return quartiles
}

func takeMinutes(takes []annotatedTake) []int {
	minutes := make([]int, 0, len(takes))
	for _, take := range takes {
		minutes = append(minutes, take.Minute)
	}

	return minutes
}
