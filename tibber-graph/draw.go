package tibbergraph

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

const (
	// X-axis starting point for drawing
	DRAW_START_X = 16

	// Y-axis thresholds for pixel changes
	THRESHOLD_ONE_PIXEL    = 0.1000
	THRESHOLD_TWO_PIXELS   = 0.3000
	THRESHOLD_THREE_PIXELS = 0.5000

	// Threshold for large price difference
	THRESHOLD_LARGE_DIFFERENCE = 0.5000

	// Y-axis center point
	Y_CENTER = 3
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

func createGraphPayload(currentTime time.Time, jsonData string) (GraphData, error) {
	var data struct {
		PriceInfo struct {
			Current  PriceData   `json:"current"`
			Today    []PriceData `json:"today"`
			Tomorrow []PriceData `json:"tomorrow"`
		} `json:"priceInfo"`
	}

	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return GraphData{}, err
	}

	allData := append(data.PriceInfo.Today, data.PriceInfo.Tomorrow...)
	sortPriceData(allData)

	currentIndex := findCurrentPriceIndex(allData, currentTime)
	graphData := GraphData{}
	currentPrice := data.PriceInfo.Current.Total

	// Draw past 3 hours
	for i := max(0, currentIndex-3); i < currentIndex; i++ {
		graphData.Draw = append(graphData.Draw, createDrawCommand(i-currentIndex+DRAW_START_X, allData[i].Total, currentPrice, "#999999"))
	}

	// Draw current hour
	graphData.Draw = append(graphData.Draw, createDrawCommand(DRAW_START_X, currentPrice, currentPrice, "#0000FF"))

	// Draw future hours
	for i := currentIndex + 1; i < min(len(allData), currentIndex+16); i++ {
		color := "#00FF00"
		if allData[i].Total > currentPrice {
			color = "#FF0000"
		}
		graphData.Draw = append(graphData.Draw, createDrawCommand(i-currentIndex+DRAW_START_X, allData[i].Total, currentPrice, color))
	}

	return graphData, nil
}

func createDrawCommand(x int, price, currentPrice float64, color string) DrawCommand {
	y := calculateY(price, currentPrice)
	return DrawCommand{DP: [3]interface{}{x, y, color}}
}

func calculateY(price, currentPrice float64) int {
	diff := price - currentPrice
	absDiff := math.Abs(diff)
	y := Y_CENTER

	if absDiff < THRESHOLD_ONE_PIXEL {
		y -= int(math.Copysign(1, diff))
	} else if absDiff < THRESHOLD_TWO_PIXELS {
		y -= int(math.Copysign(2, diff))
	} else {
		y -= int(math.Copysign(3, diff))
		if absDiff > THRESHOLD_LARGE_DIFFERENCE {
			return y
		}
	}

	return y
}

func sortPriceData(data []PriceData) {
	// Implement sorting logic here
	// For this example, we assume the data is already sorted
}

func findCurrentPriceIndex(data []PriceData, currentTime time.Time) int {
	for i, d := range data {
		if d.StartsAt.After(currentTime) {
			return i - 1
		}
	}
	return len(data) - 1
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

func printDataMatrix(data GraphData) {
	matrix := make([][]string, 8)
	for i := range matrix {
		matrix[i] = make([]string, 32)
		for j := range matrix[i] {
			matrix[i][j] = "."
		}
	}

	for _, cmd := range data.Draw {
		x := cmd.DP[0].(int)
		y := cmd.DP[1].(int)
		color := cmd.DP[2].(string)
		symbol := getSymbolForColor(color)
		matrix[y][x] = symbol
	}

	// Print column numbers
	fmt.Print("   ")
	for i := 0; i < 32; i++ {
		fmt.Printf("%d", i/10)
	}
	fmt.Print("\n   ")
	for i := 0; i < 32; i++ {
		fmt.Printf("%d", i%10)
	}
	fmt.Println()

	// Print matrix with row numbers
	for i, row := range matrix {
		fmt.Printf("%d: ", i)
		for _, cell := range row {
			fmt.Print(cell)
		}
		fmt.Println()
	}

	// Print legend
	fmt.Println("\nLegend:")
	fmt.Println("# - Past (Gray)")
	fmt.Println("O - Current (Blue)")
	fmt.Println("+ - Future, Lower Price (Green)")
	fmt.Println("X - Future, Higher Price (Red)")
	fmt.Printf("* - Large Price Difference (White, >%.4f)\n", THRESHOLD_LARGE_DIFFERENCE)
	fmt.Println(". - Empty")
}

func getSymbolForColor(color string) string {
	switch color {
	case "#999999":
		return "#" // Past (Gray)
	case "#0000FF":
		return "O" // Current (Blue)
	case "#00FF00":
		return "+" // Future, Lower Price (Green)
	case "#FF0000":
		return "X" // Future, Higher Price (Red)
	case "#FFFFFF":
		return "*" // Large Price Difference (White)
	default:
		return "?"
	}
}

func (g *GraphData) PrintDataMatrix() {
	printDataMatrix(*g)
}

func (g *GraphData) GetJson() (string, error) {
	j, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func CreateDraw(jsonData string, currentTime time.Time) (graph GraphData, err error) {
	graphData, err := createGraphPayload(currentTime, jsonData)
	if err != nil {
		return GraphData{}, err
	}
	return graphData, nil
}
