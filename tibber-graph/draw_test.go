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
			gotGraph, err := CreateDraw(tt.args.jsonData, tt.args.currentTime)

			gotGraph.PrintDataMatrix()

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDraw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(gotGraph.Draw) < tt.wantGraphDrawnPixelGreaterThan {
				t.Errorf("CreateDraw() = %v, want %v", len(gotGraph.Draw), tt.wantGraphDrawnPixelGreaterThan)
			}
		})
	}
}
