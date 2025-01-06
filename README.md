[![Go](https://github.com/dhcgn/go-mqtt-dispatcher/actions/workflows/go.yml/badge.svg)](https://github.com/dhcgn/go-mqtt-dispatcher/actions/workflows/go.yml)
[![codecov](https://codecov.io/github/dhcgn/go-mqtt-dispatcher/graph/badge.svg?token=76wV1FyxEw)](https://codecov.io/github/dhcgn/go-mqtt-dispatcher)
[![Alt](https://img.shields.io/badge/Docker-2CA5E0?style=for-the-badge&logo=docker&logoColor=white)](https://hub.docker.com/r/dhcgn/go-mqtt-dispatcher)

# go-mqtt-dispatcher

A lightweight Go application that transforms MQTT messages between devices. It subscribes to source topics, processes numeric data using JSONPath extraction and formatting, and republishes the transformed messages to target topics compatible to **Awtrix 3**.

Key features:

- Transforms numeric MQTT payloads using JSONPath
- Supports message accumulation from multiple sources
- Formats output with customizable templates
- Designed for IoT dashboard displays like Awtrix 3
- Works with any MQTT-capable device as data source
- Data can also be fetched via HTTP in a set interval

Nice to have features:

- Support of color functions depending on the value
- Support of icons stored on the device

Ideal for:

- Converting raw sensor data to human-readable formats
- Aggregating values from multiple IoT devices
- Creating formatted displays from MQTT data streams
- Real-time data transformation pipelines

```yaml
# Example: Transform temperature reading
Input:  {"sensor": {"temp": 23.456}}
Output: {"text": "23°C"} # Format of Awtrix 3
```

Docker image available for easy deployment.

## Stats

![Alt](https://repobeats.axiom.co/api/embed/b1afc9cf8420a60d30dabbbfef1f245e67193a02.svg "Repobeats analytics image")


## Personal Demonstration

### Product

![Product demonstration](README.md_assets/product.jpg)

### Wattage of the house

![House wattage display](README.md_assets/house.jpeg)

### Solar power

![Solar power display](README.md_assets/solar.jpeg)

### Tibber price (price per kWh in this hour)

![Tibber price display](README.md_assets/tibber_price.jpeg)

### Tibber price graph

![Tibber price graph display](README.md_assets/tibber_price_graph.jpeg)

Legende:

- Grey: the past three hours
- Blue: the current hour
- Yellow: the next hour if more expensive than the previous hour
- Green: the next hour if cheaper than the previous hour
- Red: highest price the this graph

## Use case

Send data to the [awtrix 3 mqtt api](https://blueforcer.github.io/awtrix3/#/api?id=example-1), the current feature [mqtt-placeholder](https://blueforcer.github.io/awtrix3/#/api?id=mqtt-placeholder) is not sufficient for all use cases.

Shelly publishes on mqtt `{"total_act_power": 392.572}` and this app transforms it to `392 W` and sends it to the awtrix 3 device trough mqtt.

## Current limitations

- Only works with numeric values
- No authentication to mqtt
- Only protocol `mqtt://` is supported
- No TLS support

## Sequence Diagram

```mermaid
sequenceDiagram
    participant Shelly
    participant MQTT Broker
    participant Go App
    participant Awtrix3

    Shelly->>MQTT Broker: Publish {"total_act_power": 392.572}
    MQTT Broker->>Go App: Forward message
    Note over Go App: Transform:<br/>- Round to integer<br/>- Add "W" unit
    Go App->>MQTT Broker: Publish "392 W"
    MQTT Broker->>Awtrix3: Display "392 W"
```

### Accumulation Sequence Diagram

```mermaid
sequenceDiagram
    participant Solar Inverter 1
    participant Solar Inverter 2
    participant MQTT Broker
    participant Go App
    participant Awtrix3

    Note over Go App: Initialize state map<br/>for "Solar" group
    Solar Inverter 1->>MQTT Broker: Publish {"apower": 1500}
    MQTT Broker->>Go App: Forward message
    Note over Go App: Store value 1500<br/>for Solar Inverter 1
    Solar Inverter 2->>MQTT Broker: Publish {"apower": 300}
    MQTT Broker->>Go App: Forward message
    Note over Go App: Store value -300<br/>for Solar Inverter 2 (inverted beacuse of negative power)
    Note over Go App: Sum values:<br/>1500 + (-300) = 1200
    Go App->>MQTT Broker: Publish "1200 W"
    MQTT Broker->>Awtrix3: Display "1200 W"
```

## Example

### Source

> From a Shelly Pro 3EM shellypro3em-00000000000 the value of the json property `"total_act_power"` from the topic `.../status/em:0`.

Topic: `shellies/shellypro3em-00000000000/status/em:0`

Payload:

```json
{
    "id": 0,
    "a_current": 0.595,
    "a_voltage": 225.3,
    "a_act_power": -60.2,
    "a_aprt_power": 134.0,
    "a_pf": 0.45,
    "a_freq": 50.0,
    "b_current": 1.041,
    "b_voltage": 223.9,
    "b_act_power": 72.8,
    "b_aprt_power": 233.1,
    "b_pf": 0.32,
    "b_freq": 50.0,
    "c_current": 2.057,
    "c_voltage": 224.3,
    "c_act_power": 379.9,
    "c_aprt_power": 461.5,
    "c_pf": 0.82,
    "c_freq": 50.0,
    "n_current": null,
    "total_current": 3.693,
    "total_act_power": 392.572,
    "total_aprt_power": 828.614,
    "user_calibrated_phase": []
}
```


### Target

> To a awtrix 3 device, a custom app `house power` with the text `392 W`.

Topic: `awtrix_b77810/custom/house power`

```json
{
  "text": "392 W",
  "icon": "redplug" // optional
}
```

`redplug` is the icon name for a red power plug, which must be uploaded to the awtrix 3 device. 

### Config

```yaml
# check with 
mqtt:
  broker: mqtt://192.168.3.10:1883

dispatcher-entries:
  - name: "Shelly Pro 3EM to topic house power"
    source:
      mqtt:
        topics-to-subscribe: 
          - topic: "shellies/shellypro3em-00000000000000000/status/em:0"
            transform:
              jsonPath: "$.total_act_power"
    topics-to-publish: 
      - topic: "awtrix_demo/custom/house power"
        transform:
          outputFormat: "%.0f W"
    icon: "redplug"
    color-script: |
      function get_color(v) {
        if (v < 500.0) {
          return "#32a852"; // green
        } else if (v < 750.0) {
          return "#FFFFFF"; // white
        } else if (v < 900.0) {
          return "#FFFF00"; // yellow
        } else if (v < 1200.0) {
          return "#FF0000"; // red
        } else {
          return "#FFC0CB"; // pink
        }
      }

  - name: "Solar power to topic solar power"
    source:
      mqtt:
        topics-to-subscribe: 
          - topic: "shellies/shellypro4pm-00000000000000/status/switch:2"
            transform:
              jsonPath: "$.apower"
              invert: true
          - topic: "shellies/shellyplusplugs-000000000000000/status/switch:0"
            transform:
              jsonPath: "$.apower"
    topics-to-publish: 
      - topic: "awtrix_demo/custom/solar power"
        transform:
          outputFormat: "%.0f W"
        filter:       
          ignore-less-than: 2.0
    icon: "ani_sun"
    operation: "sum"

    color-script: |
      function get_color(v) {
        if (v < 100.0) {
          return "#FFFFFF"; // white
        } else if (v < 250.0) {
          return "#FFA500"; // orange
        } else if (v < 500.0) {
          return "#FFFF00"; // yellow
        } else if (v < 750.0) {
          return "#008000"; // green
        } else {
          return "#FFC0CB"; // pink
        }
      }

  - name: "Tibber price from http to topic tibber price"
    diabled: true
    source:
      http:
        urls:
          - url: https://iot.hdev.io/d/00000000000000000000000000000000000000000000000000/plain-from-base64url/tibber_base64
            transform:
              jsonPath: "$.data.viewer.homes[0].currentSubscription.priceInfo.current.total"
        interval_sec: 60
    topics-to-publish:
      - topic: "awtrix_demo/custom/tibber price"
        transform:
          outputFormat: "%.4f"
    icon: "tibber"
    color-script: |
      function get_color(v) {
        if (v < 0.20) {
          return "#32a852"; // green
        } else if (v < 0.30) {
          return "#FFFFFF"; // white
        } else if (v < 0.40) {
          return "#FFFF00"; // yellow
        } else if (v < 0.50) {
          return "#FF0000"; // red
        } else {
          return "#FFC0CB"; // pink
        }
      }

  - name: "Tibber price from http to topic tibber price (only current price)"
    diabled: true
    source:
      http:
        urls:
          - url: https://iot.hdev.io/d/00000000000000000000000000000000000000000000000000/plain/priceInfo/current
        interval_sec: 60
    topics-to-publish:
      - topic: "awtrix_demo/custom/tibber price current"
        transform:
          outputFormat: "%.4f"
    icon: "tibber"
    color-script: |
      function get_color(v) {
        if (v < 0.20) {
          return "#32a852"; // green
        } else if (v < 0.30) {
          return "#FFFFFF"; // white
        } else if (v < 0.40) {
          return "#FFFF00"; // yellow
        } else if (v < 0.50) {
          return "#FF0000"; // red
        } else {
          return "#FFC0CB"; // pink
        }
      }

  - name: "Tibber price from http to topic tibber price"
    diabled: true
    source:
      http:
        urls:
          - url: https://iot.hdev.io/d/00000000000000000000000000000000000000000000000000/plain-from-base64url/tibber_base64
            transform:              
              jsonPath: "$.data.viewer.homes[0].currentSubscription"
        interval_sec: 60
    topics-to-publish:
      - topic: "awtrix_demo/custom/tibber price graph"
        transform:
          output-as-tibber-graph: true

  - name: "Tibber price as graph with Tibber API"
    disabled: false
    source:
      tibber-api:
        tibber-api-key: MY_TIBBER_API_KEY
        graphql-query: |
          {
            viewer {
              homes {
                currentSubscription {
                  priceInfo {
                    current {
                      total
                      energy
                      tax
                      startsAt
                    }
                    today {
                      total
                      energy
                      tax
                      startsAt
                    }
                    tomorrow {
                      total
                      energy
                      tax
                      startsAt
                    }
                  }
                }
              }
            }
          }
        interval_sec: 60
        transform:              
          jsonPath: "$.data.viewer.homes[0].currentSubscription"
    topics-to-publish:
      - topic: "awtrix_demo/custom/tibber price"
        transform:
          output-as-tibber-graph: true


  - name: "Tibber price with Tibber API"
    disabled: false
    source:
      tibber-api:
        tibber-api-key: MY_TIBBER_API_KEY
        graphql-query: |
          {
            viewer {
              homes {
                currentSubscription {
                  priceInfo {
                    current {
                      total
                      energy
                      tax
                      startsAt
                    }
                  }
                }
              }
            }
          }
        interval_sec: 60
        transform:              
          jsonPath: "$.data.viewer.homes[0].currentSubscription.priceInfo.current.total"
    topics-to-publish:
      - topic: "awtrix_demo/custom/tibber price"
        transform:
          outputFormat: "%.4f"
    icon: "tibber"
    color-script: |
      function get_color(v) {
        if (v < 0.20) {
          return "#32a852"; // green
        } else if (v < 0.30) {
          return "#FFFFFF"; // white
        } else if (v < 0.40) {
          return "#FFFF00"; // yellow
        } else if (v < 0.50) {
          return "#FF0000"; // red
        } else {
          return "#FFC0CB"; // pink
        }
      }
```

# MQTT Dispatcher

## Running with Docker

Run the container by mounting your config file as a volume:

```bash
docker run -v /path/to/config.yaml:/app/config.yaml dhcgn/mqtt-dispatcher:latest
```

The config file must be an absolute path!