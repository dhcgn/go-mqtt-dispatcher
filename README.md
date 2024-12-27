# go-mqtt-dispatcher

This is a simple MQTT dispatcher written in Go. It listens to a MQTT topic, processes the payload and sends it to another topic.

Use case: Send data to the [awtrix 3 mqtt api](https://blueforcer.github.io/awtrix3/#/api?id=example-1), the current feature [mqtt-placeholder](https://blueforcer.github.io/awtrix3/#/api?id=mqtt-placeholder) is not sufficient for all use cases.

## Source

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


## Target

> To a awtrix 3 device, a custom app `house power` with the text `392 W`.

Topic: `awtrix_b77810/custom/house power`

```json
{
  "text": "392 W"
}
```

### JSON Properties

> [Source](https://github.com/Blueforcer/awtrix3/blob/4a1ad2f38198cdc89733a18a97bd6079ee286615/docs/api.md?plain=1#L166)

Below are the properties you can utilize in the JSON object. **All keys are optional**; only include the properties you require.


| Key              | Type                        | Description                                                                                                                                                                  | Default | Custom App | Notification |
| ---------------- | --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ---------- | ------------ |
| `text`           | string                      | The text to display. Keep in mind the font does not have a fixed size and `I` uses less space than `W`. This facts affects when text will start scrolling                    | N/A     | X          | X            |
| `textCase`       | integer                     | Changes the Uppercase setting. 0=global setting, 1=forces uppercase; 2=shows as it sent.                                                                                     | 0       | X          | X            |
| `topText`        | boolean                     | Draw the text on top.                                                                                                                                                        | false   | X          | X            |
| `textOffset`     | integer                     | Sets an offset for the x position of a starting text.                                                                                                                        | 0       | X          | X            |
| `center`         | boolean                     | Centers a short, non-scrollable text.                                                                                                                                        | true    | X          | X            |
| `color`          | string or array of integers | The text, bar or line color.                                                                                                                                                 | N/A     | X          | X            |
| `gradient`       | Array of string or integers | Colorizes the text in a gradient of two given colors                                                                                                                         | N/A     | X          | X            |
| `blinkText`      | Integer                     | Blinks the text in an given interval in ms, not compatible with gradient or rainbow                                                                                          | N/A     | X          | X            |
| `fadeText`       | Integer                     | Fades the text on and off in an given interval, not compatible with gradient or rainbow                                                                                      | N/A     | X          | X            |
| `background`     | string or array of integers | Sets a background color.                                                                                                                                                     | N/A     | X          | X            |
| `rainbow`        | boolean                     | Fades each letter in the text differently through the entire RGB spectrum.                                                                                                   | false   | X          | X            |
| `icon`           | string                      | The icon ID or filename (without extension) to display on the app. You can also send a **8x8 jpg** as Base64 String                                                          | N/A     | X          | X            |
| `pushIcon`       | integer                     | 0 = Icon doesn't move. 1 = Icon moves with text and will not appear again. 2 = Icon moves with text but appears again when the text starts to scroll again.                  | 0       | X          | X            |
| `repeat`         | integer                     | Sets how many times the text should be scrolled through the matrix before the app ends.                                                                                      | -1      | X          | X            |
| `duration`       | integer                     | Sets how long the app or notification should be displayed.                                                                                                                   | 5       | X          | X            |
| `hold`           | boolean                     | Set it to true, to hold your **notification** on top until you press the middle button or dismiss it via HomeAssistant. This key only belongs to notification.               | false   |            | X            |
| `sound`          | string                      | The filename of your RTTTL ringtone file placed in the MELODIES folder (without extension). Or the 4 digit number of your MP3 if youre using a DFplayer                      | N/A     |            | X            |
| `rtttl`          | string                      | Allows to send the RTTTL sound string with the json.                                                                                                                         | N/A     |            | X            |
| `loopSound`      | boolean                     | Loops the sound or rtttl as long as the notification is running.                                                                                                             | false   |            | X            |
| `bar`            | array of integers           | Draws a bargraph. Without icon maximum 16 values, with icon 11 values.                                                                                                       | N/A     | X          | X            |
| `line`           | array of integers           | Draws a linechart. Without icon maximum 16 values, with icon 11 values.                                                                                                      | N/A     | X          | X            |
| `autoscale`      | boolean                     | Enables or disables autoscaling for bar and linechart.                                                                                                                       | true    | X          | X            |
| `barBC`          | string or array of integers | Backgroundcolor of the bars.                                                                                                                                                 | 0       | X          | X            |
| `progress`       | integer                     | Shows a progress bar. Value can be 0-100.                                                                                                                                    | -1      | X          | X            |
| `progressC`      | string or array of integers | The color of the progress bar.                                                                                                                                               | -1      | X          | X            |
| `progressBC`     | string or array of integers | The color of the progress bar background.                                                                                                                                    | -1      | X          | X            |
| `pos`            | integer                     | Defines the position of your custom page in the loop, starting at 0 for the first position. This will only apply with your first push. This function is experimental.        | N/A     | X          |              |
| `draw`           | array of objects            | Array of drawing instructions. Each object represents a drawing command. See the drawing instructions below.                                                                 |         | X          | X            |
| `lifetime`       | integer                     | Removes the custom app when there is no update after the given time in seconds.                                                                                              | 0       | X          |              |
| `lifetimeMode`   | integer                     | 0 = deletes the app, 1 = marks it as staled with a red rectangle around the app                                                                                              | 0       | X          |              |
| `stack`          | boolean                     | Defines if the **notification** will be stacked. `false` will immediately replace the current notification.                                                                  | true    |            | X            |
| `wakeup`         | boolean                     | If the Matrix is off, the notification will wake it up for the time of the notification.                                                                                     | false   |            | X            |
| `noScroll`       | boolean                     | Disables the text scrolling.                                                                                                                                                 | false   | X          | X            |
| `clients`        | array of strings            | Allows forwarding a notification to other awtrix devices. Use the MQTT prefix for MQTT and IP addresses for HTTP.                                                            |         |            | X            |
| `scrollSpeed`    | integer                     | Modifies the scroll speed. Enter a percentage value of the original scroll speed.                                                                                            | 100     | X          | X            |
| `effect`         | string                      | Shows an [effect](https://blueforcer.github.io/awtrix3/#/effects) as background.The effect can be removed by sending an empty string for effect                              |         | X          | X            |
| `effectSettings` | json map                    | Changes color and speed of the [effect](https://blueforcer.github.io/awtrix3/#/effects).                                                                                     |         | X          | X            |
| `save`           | boolean                     | Saves your custom app into flash and reloads it after boot. Avoid this for custom apps with high update frequencies because the ESP's flash memory has limited write cycles. |         | X          |              |
| `overlay`        | string                      | Sets an effect overlay (cannot be used with global overlays).  

## Config

```yaml
mqtt:
  broker: mqtt://mqtt-broker:1883
  username: "user"
  password: "pass"

topics:
  - subscribe: "shellies/shellypro3em-00000000000/status/em:0"
    transform:
      jsonPath: "$.total_act_power"
      round: toInteger
      outputFormat: "%v W"
    publish: "awtrix_b77810/custom/house power"
```

# MQTT Dispatcher

## Running with Docker

Run the container by mounting your config file as a volume:
```
docker run -v /path/to/config.yaml:/app/config.yaml mqtt-dispatcher
```