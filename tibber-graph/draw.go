package tibbergraph

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

const (
	// Shift the entire graph a bit to the right so that up to 3 past hours are visible.
	DRAW_START_X = 3
	Y_CENTER     = 3
)

// PriceData represents a single hour’s price info.
type PriceData struct {
	Total    float64   `json:"total"`
	StartsAt time.Time `json:"startsAt"`
}

// GraphData holds all draw commands for ASCII matrix or JSON output.
type GraphData struct {
	Draw []DrawCommand `json:"draw"`
}

// DrawCommand is a single “point” or symbol in our ASCII matrix.
type DrawCommand struct {
	DP [3]interface{} `json:"dp"`
}

// PriceInfo wraps current + today + tomorrow slices.
type PriceInfo struct {
	Current  PriceData   `json:"current"`
	Today    []PriceData `json:"today"`
	Tomorrow []PriceData `json:"tomorrow"`
}

// parsePriceData pulls the relevant portion of JSON into our struct.
func parsePriceData(jsonData string) (PriceInfo, error) {
	var data struct {
		PriceInfo PriceInfo `json:"priceInfo"`
	}
	err := json.Unmarshal([]byte(jsonData), &data)
	return data.PriceInfo, err
}

// CreateDraw combines past/current/future hours into a set of draw commands
// and maps them onto a 32x8 ASCII matrix. Missing hours are simply skipped.
func CreateDraw(jsonData string, currentTime time.Time) (GraphData, error) {
	pi, err := parsePriceData(jsonData)
	if err != nil {
		return GraphData{}, err
	}

	// Combine Today and Tomorrow into one slice
	allData := append(pi.Today, pi.Tomorrow...)
	sortPriceData(allData)

	// Find index of current hour in sorted data
	currentIndex := findCurrentPriceIndex(allData, currentTime)
	if currentIndex < 0 {
		currentIndex = 0
	} else if currentIndex >= len(allData) {
		currentIndex = len(allData) - 1
	}

	// Find extremes for y-axis mapping
	lowestPrice := findLowestPrice(allData)
	highestPrice := findHighestPrice(allData)
	rangePrice := highestPrice - lowestPrice
	if rangePrice <= 0 {
		rangePrice = 1.0 // avoid div-by-zero if all prices are identical
	}

	// Helper closure to map price -> row [0..7], top=highest, bottom=lowest
	mapPriceToY := func(p float64) int {
		scaled := 7.0 - ((p - lowestPrice) / rangePrice * 7.0)
		y := int(scaled + 0.5) // round
		if y < 0 {
			y = 0
		} else if y > 7 {
			y = 7
		}
		return y
	}

	var graph GraphData
	currentPrice := pi.Current.Total

	//----------------------------------------------------------------------
	// 1) Draw up to 3 past hours (gray)
	//----------------------------------------------------------------------
	startPast := max(0, currentIndex-3)
	for i := startPast; i < currentIndex; i++ {
		x := (i - currentIndex) + DRAW_START_X
		if x < 0 || x >= 32 {
			continue // skip if out of 32-column range
		}
		y := mapPriceToY(allData[i].Total)
		graph.Draw = append(graph.Draw, DrawCommand{DP: [3]interface{}{x, y, "#999999"}})
	}

	//----------------------------------------------------------------------
	// 2) Draw the current hour (blue)
	//----------------------------------------------------------------------
	if currentIndex >= 0 && currentIndex < len(allData) {
		if DRAW_START_X >= 0 && DRAW_START_X < 32 {
			y := mapPriceToY(currentPrice)
			graph.Draw = append(graph.Draw, DrawCommand{DP: [3]interface{}{DRAW_START_X, y, "#0000FF"}})
		}
	}

	//----------------------------------------------------------------------
	// 3) Draw future hours (no placeholders for missing ones)
	//----------------------------------------------------------------------
	for i := currentIndex + 1; i < len(allData); i++ {
		x := (i - currentIndex) + DRAW_START_X
		if x < 0 || x >= 32 {
			continue // just skip out-of-range columns
		}
		futurePrice := allData[i].Total
		y := mapPriceToY(futurePrice)

		color := "#00FF00" // default green
		// If this price is higher than the previous hour’s price, mark yellow
		if i > 0 && futurePrice > allData[i-1].Total {
			color = "#FFFF00"
		}
		// If it's the absolute highest, mark red
		if futurePrice == highestPrice {
			color = "#FF0000"
		}

		graph.Draw = append(graph.Draw, DrawCommand{DP: [3]interface{}{x, y, color}})
	}

	return graph, nil
}

// sortPriceData sorts by StartsAt ascending.
func sortPriceData(data []PriceData) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].StartsAt.Before(data[j].StartsAt)
	})
}

// findCurrentPriceIndex returns the index of the hour that is current
// or the last hour before currentTime. If all are after currentTime, returns 0 or -1.
func findCurrentPriceIndex(data []PriceData, currentTime time.Time) int {
	for i, d := range data {
		if d.StartsAt.After(currentTime) {
			if i == 0 {
				return 0
			}
			return i - 1
		}
	}
	return len(data) - 1
}

// findLowestPrice scans all data for min price.
func findLowestPrice(data []PriceData) float64 {
	if len(data) == 0 {
		return 0
	}
	lowest := data[0].Total
	for _, d := range data {
		if d.Total < lowest {
			lowest = d.Total
		}
	}
	return lowest
}

// findHighestPrice scans all data for max price.
func findHighestPrice(data []PriceData) float64 {
	var highest float64
	for _, d := range data {
		if d.Total > highest {
			highest = d.Total
		}
	}
	return highest
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// PrintDataMatrix draws an 8 (rows) x 32 (columns) ASCII matrix.
func (g *GraphData) PrintDataMatrix() {
	const rows, cols = 8, 32

	// Initialize matrix with "." everywhere
	matrix := make([][]string, rows)
	for i := 0; i < rows; i++ {
		matrix[i] = make([]string, cols)
		for j := 0; j < cols; j++ {
			matrix[i][j] = "."
		}
	}

	// Place each DP in the matrix (if in range)
	for _, cmd := range g.Draw {
		x := toInt(cmd.DP[0])
		y := toInt(cmd.DP[1])
		color := cmd.DP[2].(string)
		symbol := getSymbolForColor(color)
		if x >= 0 && x < cols && y >= 0 && y < rows {
			matrix[y][x] = symbol
		}
	}

	// Print column header
	fmt.Print("   ")
	for col := 0; col < cols; col++ {
		fmt.Printf("%d", col/10)
	}
	fmt.Print("\n   ")
	for col := 0; col < cols; col++ {
		fmt.Printf("%d", col%10)
	}
	fmt.Println()

	// Print rows
	for r := 0; r < rows; r++ {
		fmt.Printf("%d: ", r)
		for c := 0; c < cols; c++ {
			fmt.Print(matrix[r][c])
		}
		fmt.Println()
	}

	// Print legend
	fmt.Println("\nLegend:")
	fmt.Println("# - Past (Gray)")
	fmt.Println("O - Current (Blue)")
	fmt.Println("+ - Future, Lower Price (Green)")
	fmt.Println("* - Future, Higher Price (Yellow)")
	fmt.Println("X - Highest Price (Red)")
	fmt.Println(". - Empty")
}

// Convert command param to int
func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

// Map colors to ASCII symbols
func getSymbolForColor(color string) string {
	switch color {
	case "#999999":
		return "#" // Past (Gray)
	case "#0000FF":
		return "O" // Current (Blue)
	case "#00FF00":
		return "+" // Future, Lower Price (Green)
	case "#FFFF00":
		return "*" // Future, Higher Price (Yellow)
	case "#FF0000":
		return "X" // Highest Price (Red)
	default:
		return "?" // fallback
	}
}

// GetJson returns the GraphData as a pretty-printed JSON string.
func (g *GraphData) GetJson() (string, error) {
	j, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// Stats is just an extra struct for demonstration (not core to drawing).
type Stats struct {
	UsedTime time.Time

	MaxLast3Hours []PriceData
	CurrentHour   PriceData
	MaxNext5Hours []PriceData

	NumPastHours   int
	NumFutureHours int
}

// createStats is a helper if you need stats. Not strictly required for the ASCII graph.
func createStats(pi PriceInfo, currentTime time.Time) (Stats, error) {
	allData := append(pi.Today, pi.Tomorrow...)

	currentIndex := findCurrentPriceIndex(allData, currentTime)

	// Collect max past 3 hours
	startPast := max(0, currentIndex-3)
	maxLast3Hours := allData[startPast:currentIndex]

	// Collect max 5 future hours
	startFuture := currentIndex + 1
	endFuture := min(len(allData), startFuture+5)
	maxNext5Hours := allData[startFuture:endFuture]

	stats := Stats{
		UsedTime:       currentTime,
		MaxLast3Hours:  maxLast3Hours,
		CurrentHour:    pi.Current,
		MaxNext5Hours:  maxNext5Hours,
		NumPastHours:   currentIndex,
		NumFutureHours: len(allData) - currentIndex - 1,
	}

	return stats, nil
}
