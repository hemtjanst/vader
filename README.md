# Väder

Väder exposes the "feels like" temperature and relative humidity as returned bi
the Wunderground API as a temperature and a humidity sensor to HomeKit.

## Installation

The installation is pretty simple, `go install` it, get an [API token for
Wunderground](wapi) and then point it at your MQTT broker.

```
go install github.com/hemtjanst/vader/cmd/vader
vader -token XYZ -mqtt.address broker.mydomain.tld:1883
```

The free plan on Wunderground allows for 500 calls per day and 10 calls per
minute. `väder` defaults to 1 call per hour, so there's plenty left to play
around with.

[wapi]: https://www.wunderground.com/weather/api/d/pricing.html

## Configuration

It's required to pass in `-token` which has to be a [Wunderground API](wapi)
token.

By default we will refresh the weather data once per hour. It's currently not
possible to configure it to poll more often but you can decrease the update
frequency by passing in an integer bigger than 1 to `-refresh`.

See the `--help` for all possible options.

## Attribution

<img src="https://icons.wxug.com/logos/JPG/wundergroundLogo_4c_horz.jpg" height="100">
