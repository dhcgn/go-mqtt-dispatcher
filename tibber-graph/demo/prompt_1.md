Help me with the Go code.

This code doenst print all future values. In the test case I have 11 futur hours, but only 11 are printed.


got: `4: ###O+***X++++++.................` 
expceted: not `.` because every future hour should be printed.

But be aware, implement it in a robust matter, sometimes less of past or future values are available.



# All data you need

File: draw.go

```go
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

	// Combine Today and Tomorrow into one continuous slice
	allData := append(pi.Today, pi.Tomorrow...)
	sortPriceData(allData)

	currentIndex := findCurrentPriceIndex(allData, currentTime)
	currentPrice := pi.Current.Total

	highestPrice := findHighestPrice(allData)
	var graph GraphData

	//----------------------------------------------------------------------
	// 1) Draw up to 3 past hours (gray)
	//----------------------------------------------------------------------
	// If currentIndex is < 3, then we might have fewer than 3 past hours
	startPast := max(0, currentIndex-3)
	for i := startPast; i < currentIndex; i++ {
		x := (i - currentIndex) + DRAW_START_X
		if x < 0 || x >= 32 {
			continue // skip if out of screen range
		}
		graph.Draw = append(graph.Draw, createDrawCommand(x, allData[i].Total, currentPrice, "#999999"))
	}

	//----------------------------------------------------------------------
	// 2) Draw the current hour (blue)
	//----------------------------------------------------------------------
	if currentIndex >= 0 && currentIndex < len(allData) {
		// Place the current hour exactly at DRAW_START_X
		if DRAW_START_X >= 0 && DRAW_START_X < 32 {
			graph.Draw = append(
				graph.Draw,
				createDrawCommand(DRAW_START_X, currentPrice, currentPrice, "#0000FF"),
			)
		}
	}

	//----------------------------------------------------------------------
	// 3) Draw all future hours, until we run out of data or columns
	//----------------------------------------------------------------------
	for i := currentIndex + 1; i < len(allData); i++ {
		x := (i - currentIndex) + DRAW_START_X
		if x < 0 || x >= 32 {
			// Stop if we're outside the screen columns
			if x >= 32 {
				break
			}
			continue
		}

		color := "#00FF00" // default to Green for future
		// If the price is higher than the previous hour, mark it Yellow
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

func findCurrentPriceIndex(data []PriceData, currentTime time.Time) int {
	// Return the index of the hour that is "current" or last hour before currentTime
	for i, d := range data {
		if d.StartsAt.After(currentTime) {
			return i - 1
		}
	}
	return len(data) - 1
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

```

File: draw_test.go

```go
package tibbergraph

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/utils"
	"testing"
	"time"

	"github.com/oliveagle/jsonpath"
)

func transformPayloadWithJsonPath(payload []byte, json_path string) []byte {
	var json_data interface{}
	json.Unmarshal([]byte(payload), &json_data)

	res, err := jsonpath.JsonPathLookup(json_data, json_path)
	if err != nil {
		return []byte{}
	}

	j, err := json.Marshal(res)
	if err != nil {
		return []byte{}
	}

	return j
}

var testJson = `{"data":{"viewer":{"homes":[{"currentSubscription":{"priceInfo":{"current":{"total":0.3293,"energy":0.13,"tax":0.1993,"startsAt":"2025-01-04T12:00:00.000+01:00"},"today":[{"total":0.2822,"energy":0.0904,"tax":0.1918,"startsAt":"2025-01-04T00:00:00.000+01:00"},{"total":0.282,"energy":0.0903,"tax":0.1917,"startsAt":"2025-01-04T01:00:00.000+01:00"},{"total":0.2819,"energy":0.0902,"tax":0.1917,"startsAt":"2025-01-04T02:00:00.000+01:00"},{"total":0.2835,"energy":0.0915,"tax":0.192,"startsAt":"2025-01-04T03:00:00.000+01:00"},{"total":0.2887,"energy":0.096,"tax":0.1927,"startsAt":"2025-01-04T04:00:00.000+01:00"},{"total":0.2961,"energy":0.1021,"tax":0.194,"startsAt":"2025-01-04T05:00:00.000+01:00"},{"total":0.3062,"energy":0.1107,"tax":0.1955,"startsAt":"2025-01-04T06:00:00.000+01:00"},{"total":0.3173,"energy":0.12,"tax":0.1973,"startsAt":"2025-01-04T07:00:00.000+01:00"},{"total":0.3273,"energy":0.1284,"tax":0.1989,"startsAt":"2025-01-04T08:00:00.000+01:00"},{"total":0.3301,"energy":0.1308,"tax":0.1993,"startsAt":"2025-01-04T09:00:00.000+01:00"},{"total":0.3297,"energy":0.1303,"tax":0.1994,"startsAt":"2025-01-04T10:00:00.000+01:00"},{"total":0.3272,"energy":0.1283,"tax":0.1989,"startsAt":"2025-01-04T11:00:00.000+01:00"},{"total":0.3293,"energy":0.13,"tax":0.1993,"startsAt":"2025-01-04T12:00:00.000+01:00"},{"total":0.3291,"energy":0.1298,"tax":0.1993,"startsAt":"2025-01-04T13:00:00.000+01:00"},{"total":0.3369,"energy":0.1364,"tax":0.2005,"startsAt":"2025-01-04T14:00:00.000+01:00"},{"total":0.3396,"energy":0.1387,"tax":0.2009,"startsAt":"2025-01-04T15:00:00.000+01:00"},{"total":0.3435,"energy":0.142,"tax":0.2015,"startsAt":"2025-01-04T16:00:00.000+01:00"},{"total":0.3545,"energy":0.1512,"tax":0.2033,"startsAt":"2025-01-04T17:00:00.000+01:00"},{"total":0.3457,"energy":0.1438,"tax":0.2019,"startsAt":"2025-01-04T18:00:00.000+01:00"},{"total":0.3387,"energy":0.138,"tax":0.2007,"startsAt":"2025-01-04T19:00:00.000+01:00"},{"total":0.3286,"energy":0.1295,"tax":0.1991,"startsAt":"2025-01-04T20:00:00.000+01:00"},{"total":0.3207,"energy":0.1228,"tax":0.1979,"startsAt":"2025-01-04T21:00:00.000+01:00"},{"total":0.3153,"energy":0.1183,"tax":0.197,"startsAt":"2025-01-04T22:00:00.000+01:00"},{"total":0.3048,"energy":0.1095,"tax":0.1953,"startsAt":"2025-01-04T23:00:00.000+01:00"}],"tomorrow":[]}}}]}}}`
var testDateTimeLayout = "2006-01-02T15:04:05.000-07:00"
var testDateTimeDate = "2025-01-04T12:00:00.000+01:00"
var testJsonPath = "$.data.viewer.homes[0].currentSubscription"

// Implementing the TransformSource interface
type MyTransformSource struct {
	JsonPath string
	Invert   bool
}

func (mts MyTransformSource) GetJsonPath() string {
	return mts.JsonPath
}

func (mts MyTransformSource) GetInvert() bool {
	return mts.Invert
}

// TestCreateDraw tests the CreateDraw function
// run with output as ASCII Art Graph: go test -v -timeout 30s -run ^TestCreateDraw$ go-mqtt-dispatcher/tibber-graph
func TestCreateDraw(t *testing.T) {
	trans := MyTransformSource{
		JsonPath: testJsonPath,
	}

	json := utils.TransformPayloadWithJsonPath([]byte(testJson), trans)
	testDateTime, err := time.Parse(testDateTimeLayout, testDateTimeDate)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return
	}

	type args struct {
		jsonData    string
		currentTime time.Time
	}
	tests := []struct {
		name                           string
		args                           args
		wantGraphDrawnPixelGreaterThan int
		wantErr                        bool
	}{
		{
			name: "Test 1",
			args: args{
				jsonData:    string(json),
				currentTime: testDateTime,
			},
			wantGraphDrawnPixelGreaterThan: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			pi, err := parsePriceData(tt.args.jsonData)
			if err != nil {
				t.Errorf("createStats() error = %v", err)
				return
			}

			stats, err := createStats(pi, tt.args.currentTime)
			if err != nil {
				t.Errorf("createStats() error = %v", err)
				return
			}

			// Print stats for verification
			fmt.Println("Test date:", stats.UsedTime.Format("2006-01-02 15:04"))
			fmt.Println("Max past 3 hours:")
			for _, hour := range stats.MaxLast3Hours {
				fmt.Printf("Hour: %s, Price: %.4f\n", hour.StartsAt.Format("15:04"), hour.Total)
			}
			fmt.Println("Current hour:")
			fmt.Printf("Hour: %s, Price: %.4f\n", stats.CurrentHour.StartsAt.Format("15:04"), stats.CurrentHour.Total)
			fmt.Println("Max 5 future hours:")
			for _, hour := range stats.MaxNext5Hours {
				fmt.Printf("Hour: %s, Price: %.4f\n", hour.StartsAt.Format("15:04"), hour.Total)
			}
			fmt.Printf("Number of past hours: %d\n", stats.NumPastHours)
			fmt.Printf("Number of future hours: %d\n", stats.NumFutureHours)

			gotGraph, err := CreateDraw(tt.args.jsonData, tt.args.currentTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDraw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(gotGraph.Draw) < tt.wantGraphDrawnPixelGreaterThan {
				t.Errorf("CreateDraw() = %v, want %v", len(gotGraph.Draw), tt.wantGraphDrawnPixelGreaterThan)
			}

			gotGraph.PrintDataMatrix()
		})
	}
}
```

Output of func TestCreateDraw(t *testing.T) 

```plain
=== RUN   TestCreateDraw
=== RUN   TestCreateDraw/Test_1
Test date: 2025-01-04 12:00
Max past 3 hours:
Hour: 09:00, Price: 0.3301
Hour: 10:00, Price: 0.3297
Hour: 11:00, Price: 0.3272
Current hour:
Hour: 12:00, Price: 0.3293
Max 5 future hours:
Hour: 13:00, Price: 0.3291
Hour: 14:00, Price: 0.3369
Hour: 15:00, Price: 0.3396
Hour: 16:00, Price: 0.3435
Hour: 17:00, Price: 0.3545
Number of past hours: 12
Number of future hours: 11
   00000000001111111111222222222233
   01234567890123456789012345678901
0: ................................
1: ................................
2: ................................
3: ................................
4: ###O+***X++++++.................
5: ................................
6: ................................
7: ................................

Legend:
# - Past (Gray)
O - Current (Blue)
+ - Future, Lower Price (Green)
* - Future, Higher Price (Yellow)
X - Highest Price (Red)
. - Empty
--- PASS: TestCreateDraw (0.00s)
    --- PASS: TestCreateDraw/Test_1 (0.00s)
PASS
ok      go-mqtt-dispatcher/tibber-graph 0.004s
```