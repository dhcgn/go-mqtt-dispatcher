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

type PriceData struct {
	Total    float64   `json:"total"`
	StartsAt time.Time `json:"startsAt"`
}

type GraphData struct {
	Draw []DrawCommand `json:"draw"`
}

type DrawCommand struct {
	DP [3]interface{} `json:"dp"`
}

type PriceInfo struct {
	Current  PriceData   `json:"current"`
	Today    []PriceData `json:"today"`
	Tomorrow []PriceData `json:"tomorrow"`
}

func parsePriceData(jsonData string) (PriceInfo, error) {
	var data struct {
		PriceInfo PriceInfo `json:"priceInfo"`
	}
	err := json.Unmarshal([]byte(jsonData), &data)
	return data.PriceInfo, err
}

func CreateDraw(jsonData string, currentTime time.Time) (GraphData, error) {
	pi, err := parsePriceData(jsonData)
	if err != nil {
		return GraphData{}, err
	}

	// Combine Today and Tomorrow into one slice
	allData := append(pi.Today, pi.Tomorrow...)
	sortPriceData(allData)

	currentIndex := findCurrentPriceIndex(allData, currentTime)
	// Clamp currentIndex to valid range
	if currentIndex < 0 {
		currentIndex = 0
	} else if currentIndex >= len(allData) {
		currentIndex = len(allData) - 1
	}

	currentPrice := pi.Current.Total
	highestPrice := findHighestPrice(allData)

	var graph GraphData

	//----------------------------------------------------------------------
	// 1) Draw up to 3 past hours (gray)
	//----------------------------------------------------------------------
	startPast := max(0, currentIndex-3)
	for i := startPast; i < currentIndex; i++ {
		x := (i - currentIndex) + DRAW_START_X
		// Skip if out of range, but do not break
		if x < 0 || x >= 32 {
			continue
		}
		graph.Draw = append(
			graph.Draw,
			createDrawCommand(x, allData[i].Total, currentPrice, "#999999"),
		)
	}

	//----------------------------------------------------------------------
	// 2) Draw the current hour (blue)
	//----------------------------------------------------------------------
	if currentIndex >= 0 && currentIndex < len(allData) {
		if DRAW_START_X >= 0 && DRAW_START_X < 32 {
			graph.Draw = append(
				graph.Draw,
				createDrawCommand(DRAW_START_X, currentPrice, currentPrice, "#0000FF"),
			)
		}
	}

	//----------------------------------------------------------------------
	// 3) Draw all *actual* future hours (no missing placeholders)
	//----------------------------------------------------------------------
	for i := currentIndex + 1; i < len(allData); i++ {
		x := (i - currentIndex) + DRAW_START_X
		// Skip if out of the 32-column range
		if x < 0 || x >= 32 {
			continue
		}

		// Default color for future is Green
		color := "#00FF00"
		// If this price is higher than the *previous hour*, mark it Yellow
		if i > 0 && allData[i].Total > allData[i-1].Total {
			color = "#FFFF00"
		}
		// If it's the absolute highest price, mark it Red
		if allData[i].Total == highestPrice {
			color = "#FF0000"
		}

		graph.Draw = append(
			graph.Draw,
			createDrawCommand(x, allData[i].Total, currentPrice, color),
		)
	}

	return graph, nil
}

// Make findCurrentPriceIndex more defensive if everything is after currentTime
func findCurrentPriceIndex(data []PriceData, currentTime time.Time) int {
	for i, d := range data {
		if d.StartsAt.After(currentTime) {
			// If *all* data is after currentTime, i - 1 will be -1.
			// Return 0 or -1 here, depending on your preference:
			if i == 0 {
				return 0
			}
			return i - 1
		}
	}
	return len(data) - 1
}

// createDrawCommand is a small helper to build a DrawCommand at (x, y) with the given color.
func createDrawCommand(x int, price, currentPrice float64, color string) DrawCommand {
	y := calculateY(price, currentPrice)
	return DrawCommand{DP: [3]interface{}{x, y, color}}
}

// For a simple horizontal "line," keep Y = 4 for all data points.
func calculateY(price, currentPrice float64) int {
	return 4
}

func sortPriceData(data []PriceData) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].StartsAt.Before(data[j].StartsAt)
	})
}

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
// Each DrawCommand is placed if it falls within (0..31)x(0..7).
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

	// Print each row
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
		return "?"
	}
}

func (g *GraphData) GetJson() (string, error) {
	j, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(j), nil
}

type Stats struct {
	UsedTime time.Time

	MaxLast3Hours []PriceData
	CurrentHour   PriceData
	MaxNext5Hours []PriceData

	NumPastHours   int
	NumFutureHours int
}

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
