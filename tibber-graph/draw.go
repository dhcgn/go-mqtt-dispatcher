package tibbergraph

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

const (
	DRAW_START_X               = 0
	THRESHOLD_ONE_PIXEL        = 0.1000
	THRESHOLD_TWO_PIXELS       = 0.3000
	THRESHOLD_THREE_PIXELS     = 0.5000
	THRESHOLD_LARGE_DIFFERENCE = 0.5000
	Y_CENTER                   = 3
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

func parsePriceData(jsonData string) (struct {
	PriceInfo struct {
		Current  PriceData   `json:"current"`
		Today    []PriceData `json:"today"`
		Tomorrow []PriceData `json:"tomorrow"`
	} `json:"priceInfo"`
}, error) {
	var data struct {
		PriceInfo struct {
			Current  PriceData   `json:"current"`
			Today    []PriceData `json:"today"`
			Tomorrow []PriceData `json:"tomorrow"`
		} `json:"priceInfo"`
	}
	err := json.Unmarshal([]byte(jsonData), &data)
	return data, err
}

func CreateDraw(jsonData string, currentTime time.Time) (graph GraphData, err error) {
	data, err := parsePriceData(jsonData)
	if err != nil {
		return GraphData{}, err
	}

	allData := append(data.PriceInfo.Today, data.PriceInfo.Tomorrow...)
	sortPriceData(allData)

	currentIndex := findCurrentPriceIndex(allData, currentTime)
	graphData := GraphData{}
	currentPrice := data.PriceInfo.Current.Total

	highestPrice := findHighestPrice(allData)

	// Draw past 3 hours
	for i := max(0, currentIndex-3); i < currentIndex; i++ {
		graphData.Draw = append(graphData.Draw, createDrawCommand(i-currentIndex+DRAW_START_X, allData[i].Total, currentPrice, "#999999"))
	}

	// Draw current hour
	graphData.Draw = append(graphData.Draw, createDrawCommand(DRAW_START_X, currentPrice, currentPrice, "#0000FF"))

	// Draw future hours
	for i := currentIndex + 1; i < min(len(allData), currentIndex+29); i++ {
		color := "#00FF00"
		if i > currentIndex+1 && allData[i].Total > allData[i-1].Total {
			color = "#FFFF00"
		}
		if allData[i].Total == highestPrice {
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
	return 4
}

func sortPriceData(data []PriceData) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].StartsAt.Before(data[j].StartsAt)
	})
}

func findCurrentPriceIndex(data []PriceData, currentTime time.Time) int {
	for i, d := range data {
		if d.StartsAt.After(currentTime) {
			return i - 1
		}
	}
	return len(data) - 1
}

func findHighestPrice(data []PriceData) float64 {
	highest := 0.0
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

func (g *GraphData) PrintDataMatrix() {
	matrix := make([][]string, 8)
	for i := range matrix {
		matrix[i] = make([]string, 32)
		for j := range matrix[i] {
			matrix[i][j] = "."
		}
	}

	for _, cmd := range g.Draw {
		x := toInt(cmd.DP[0])
		y := toInt(cmd.DP[1])
		color := cmd.DP[2].(string)
		symbol := getSymbolForColor(color)
		if x >= 0 && x < 32 && y >= 0 && y < 8 {
			matrix[y][x] = symbol
		}
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
	fmt.Println("* - Future, Higher Price (Yellow)")
	fmt.Println("X - Highest Price (Red)")
	fmt.Println(". - Empty")
}

// Helper function to convert interface{} to int
func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0 // or handle error as appropriate
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
