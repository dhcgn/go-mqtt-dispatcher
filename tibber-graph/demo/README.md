Write a Go function that creates a JSON payload which represents a graph of the price `total` over time `startsAt`.

- not always the we have the same number of data points
- not always is `tomorrow` present, because the data is not yet available
- create a struct type for the data
- create a helper class to print the data in a human readable format, like a matrix with symbols
- the function arguments are the current datetime and the JSON Data (see JSON Data), the return value is struct with the graph data (see MQTT Payload)
- combine in the json data "today" and "tomorrow" 

## Drawing Instructions

- Each pixel is one hour
- colors:
    - pixel from the past are gray #999999
    - pixel from the current hour are blue #0000FF
    - pixel from the future are green #00FF00 or red #FF0000 depending on the price on comparison with the current price
- the graph is drawn from left to right
- the graph is drawn from center to center, where center is the current price
- the graph can go up and down 3 pixels
    - one pixel is less than 0.1000 difference in price
    - two pixel is less than 0.3000 difference in price
    - three pixel is less than 0.5000 difference in price
    - make the third pixel white if the price is more than 0.5000 difference in price
- draw three pixel fo the past 3 hours, than the current hour and than the next hours until the end of data how have

## MQTT Payload

This draw a pixel at each corner of the screen.

```json
{
    "draw": [
        {
            "dp": [
                0,
                0,
                "#D61F30"
            ]
        },
        {
            "dp": [
                0,
                7,
                "#D61F30"
            ]
        },
        {
            "dp": [
                31,
                0,
                "#D61F30"
            ]
        },
        {
            "dp": [
                31,
                7,
                "#D61F30"
            ]
        }
    ]
}
```

- Command: `dp`
- Array Values: `[x, y, cl]`
- Description: Draw a pixel at position (x, y) with color cl
- Example: `{"dp": [28, 4, "#FF0000"]},`



## JSON Data

```json
{
  "priceInfo": {
    "current": {
      "total": 0.3001,
      "energy": 0.1055,
      "tax": 0.1946,
      "startsAt": "2025-01-03T16:00:00.000+01:00"
    },
    "today": [
      {
        "total": 0.293,
        "energy": 0.0996,
        "tax": 0.1934,
        "startsAt": "2025-01-03T00:00:00.000+01:00"
      },
      {
        "total": 0.2787,
        "energy": 0.0875,
        "tax": 0.1912,
        "startsAt": "2025-01-03T01:00:00.000+01:00"
      },
      {
        "total": 0.2757,
        "energy": 0.085,
        "tax": 0.1907,
        "startsAt": "2025-01-03T02:00:00.000+01:00"
      },
      {
        "total": 0.2743,
        "energy": 0.0838,
        "tax": 0.1905,
        "startsAt": "2025-01-03T03:00:00.000+01:00"
      },
      {
        "total": 0.2668,
        "energy": 0.0775,
        "tax": 0.1893,
        "startsAt": "2025-01-03T04:00:00.000+01:00"
      },
      {
        "total": 0.2696,
        "energy": 0.0799,
        "tax": 0.1897,
        "startsAt": "2025-01-03T05:00:00.000+01:00"
      },
      {
        "total": 0.274,
        "energy": 0.0836,
        "tax": 0.1904,
        "startsAt": "2025-01-03T06:00:00.000+01:00"
      },
      {
        "total": 0.2679,
        "energy": 0.0784,
        "tax": 0.1895,
        "startsAt": "2025-01-03T07:00:00.000+01:00"
      },
      {
        "total": 0.2728,
        "energy": 0.0826,
        "tax": 0.1902,
        "startsAt": "2025-01-03T08:00:00.000+01:00"
      },
      {
        "total": 0.2707,
        "energy": 0.0808,
        "tax": 0.1899,
        "startsAt": "2025-01-03T09:00:00.000+01:00"
      },
      {
        "total": 0.2669,
        "energy": 0.0776,
        "tax": 0.1893,
        "startsAt": "2025-01-03T10:00:00.000+01:00"
      },
      {
        "total": 0.2584,
        "energy": 0.0705,
        "tax": 0.1879,
        "startsAt": "2025-01-03T11:00:00.000+01:00"
      },
      {
        "total": 0.2578,
        "energy": 0.0699,
        "tax": 0.1879,
        "startsAt": "2025-01-03T12:00:00.000+01:00"
      },
      {
        "total": 0.2671,
        "energy": 0.0778,
        "tax": 0.1893,
        "startsAt": "2025-01-03T13:00:00.000+01:00"
      },
      {
        "total": 0.2683,
        "energy": 0.0788,
        "tax": 0.1895,
        "startsAt": "2025-01-03T14:00:00.000+01:00"
      },
      {
        "total": 0.276,
        "energy": 0.0853,
        "tax": 0.1907,
        "startsAt": "2025-01-03T15:00:00.000+01:00"
      },
      {
        "total": 0.3001,
        "energy": 0.1055,
        "tax": 0.1946,
        "startsAt": "2025-01-03T16:00:00.000+01:00"
      },
      {
        "total": 0.3134,
        "energy": 0.1167,
        "tax": 0.1967,
        "startsAt": "2025-01-03T17:00:00.000+01:00"
      },
      {
        "total": 0.3115,
        "energy": 0.1151,
        "tax": 0.1964,
        "startsAt": "2025-01-03T18:00:00.000+01:00"
      },
      {
        "total": 0.3113,
        "energy": 0.1149,
        "tax": 0.1964,
        "startsAt": "2025-01-03T19:00:00.000+01:00"
      },
      {
        "total": 0.3094,
        "energy": 0.1133,
        "tax": 0.1961,
        "startsAt": "2025-01-03T20:00:00.000+01:00"
      },
      {
        "total": 0.3028,
        "energy": 0.1078,
        "tax": 0.195,
        "startsAt": "2025-01-03T21:00:00.000+01:00"
      },
      {
        "total": 0.2914,
        "energy": 0.0982,
        "tax": 0.1932,
        "startsAt": "2025-01-03T22:00:00.000+01:00"
      },
      {
        "total": 0.2799,
        "energy": 0.0886,
        "tax": 0.1913,
        "startsAt": "2025-01-03T23:00:00.000+01:00"
      }
    ],
    "tomorrow": [
      {
        "total": 0.2822,
        "energy": 0.0904,
        "tax": 0.1918,
        "startsAt": "2025-01-04T00:00:00.000+01:00"
      },
      {
        "total": 0.282,
        "energy": 0.0903,
        "tax": 0.1917,
        "startsAt": "2025-01-04T01:00:00.000+01:00"
      },
      {
        "total": 0.2819,
        "energy": 0.0902,
        "tax": 0.1917,
        "startsAt": "2025-01-04T02:00:00.000+01:00"
      },
      {
        "total": 0.2835,
        "energy": 0.0915,
        "tax": 0.192,
        "startsAt": "2025-01-04T03:00:00.000+01:00"
      },
      {
        "total": 0.2887,
        "energy": 0.096,
        "tax": 0.1927,
        "startsAt": "2025-01-04T04:00:00.000+01:00"
      },
      {
        "total": 0.2961,
        "energy": 0.1021,
        "tax": 0.194,
        "startsAt": "2025-01-04T05:00:00.000+01:00"
      },
      {
        "total": 0.3062,
        "energy": 0.1107,
        "tax": 0.1955,
        "startsAt": "2025-01-04T06:00:00.000+01:00"
      },
      {
        "total": 0.3173,
        "energy": 0.12,
        "tax": 0.1973,
        "startsAt": "2025-01-04T07:00:00.000+01:00"
      },
      {
        "total": 0.3273,
        "energy": 0.1284,
        "tax": 0.1989,
        "startsAt": "2025-01-04T08:00:00.000+01:00"
      },
      {
        "total": 0.3301,
        "energy": 0.1308,
        "tax": 0.1993,
        "startsAt": "2025-01-04T09:00:00.000+01:00"
      },
      {
        "total": 0.3297,
        "energy": 0.1303,
        "tax": 0.1994,
        "startsAt": "2025-01-04T10:00:00.000+01:00"
      },
      {
        "total": 0.3272,
        "energy": 0.1283,
        "tax": 0.1989,
        "startsAt": "2025-01-04T11:00:00.000+01:00"
      },
      {
        "total": 0.3293,
        "energy": 0.13,
        "tax": 0.1993,
        "startsAt": "2025-01-04T12:00:00.000+01:00"
      },
      {
        "total": 0.3291,
        "energy": 0.1298,
        "tax": 0.1993,
        "startsAt": "2025-01-04T13:00:00.000+01:00"
      },
      {
        "total": 0.3369,
        "energy": 0.1364,
        "tax": 0.2005,
        "startsAt": "2025-01-04T14:00:00.000+01:00"
      },
      {
        "total": 0.3396,
        "energy": 0.1387,
        "tax": 0.2009,
        "startsAt": "2025-01-04T15:00:00.000+01:00"
      },
      {
        "total": 0.3435,
        "energy": 0.142,
        "tax": 0.2015,
        "startsAt": "2025-01-04T16:00:00.000+01:00"
      },
      {
        "total": 0.3545,
        "energy": 0.1512,
        "tax": 0.2033,
        "startsAt": "2025-01-04T17:00:00.000+01:00"
      },
      {
        "total": 0.3457,
        "energy": 0.1438,
        "tax": 0.2019,
        "startsAt": "2025-01-04T18:00:00.000+01:00"
      },
      {
        "total": 0.3387,
        "energy": 0.138,
        "tax": 0.2007,
        "startsAt": "2025-01-04T19:00:00.000+01:00"
      },
      {
        "total": 0.3286,
        "energy": 0.1295,
        "tax": 0.1991,
        "startsAt": "2025-01-04T20:00:00.000+01:00"
      },
      {
        "total": 0.3207,
        "energy": 0.1228,
        "tax": 0.1979,
        "startsAt": "2025-01-04T21:00:00.000+01:00"
      },
      {
        "total": 0.3153,
        "energy": 0.1183,
        "tax": 0.197,
        "startsAt": "2025-01-04T22:00:00.000+01:00"
      },
      {
        "total": 0.3048,
        "energy": 0.1095,
        "tax": 0.1953,
        "startsAt": "2025-01-04T23:00:00.000+01:00"
      }
    ]
  }
}
```