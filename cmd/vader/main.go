package main

import (
	"flag"
	"fmt"
	mq "github.com/eclipse/paho.mqtt.golang"
	"github.com/hemtjanst/hemtjanst/device"
	"github.com/hemtjanst/hemtjanst/messaging"
	"github.com/hemtjanst/hemtjanst/messaging/flagmqtt"
	"github.com/hemtjanst/vader"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	location = flag.String("location", "autoip", "Location to fetch the current conditions of")
	refresh  = flag.Int("refresh", 1, "Time in hours after which to query the Wunderground API for new data")
	apiToken = flag.String("token", "REQUIRED", "Wunderground API token")
)

type handler struct {
	devices []*device.Device
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Parameters:\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}
	flag.Parse()

	if *apiToken == "REQUIRED" {
		log.Fatal("A token is required to be able to query the Wunderground API\n")
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	id := flagmqtt.NewUniqueIdentifier()
	h := &handler{
		devices: []*device.Device{},
	}
	conf := flagmqtt.ClientConfig{
		ClientID:         "väder",
		WillTopic:        "leave",
		WillPayload:      id,
		WillRetain:       false,
		WillQoS:          0,
		OnConnectHandler: h.onConnectHandler,
	}
	c, err := flagmqtt.NewPersistentMqtt(conf)
	if err != nil {
		log.Fatal("Could not configure the MQTT client: ", err)
	}

	m := messaging.NewMQTTMessenger(c)

	tempSensor := device.NewDevice("weather/temperature", m)
	tempSensor.Manufacturer = "väder"
	tempSensor.Name = "Temperature (outside)"
	tempSensor.LastWillID = id
	tempSensor.Type = "temperatureSensor"
	tempSensor.AddFeature("currentTemperature", &device.Feature{
		Min: -50,
	})
	h.devices = append(h.devices, tempSensor)

	humiditySensor := device.NewDevice("weather/humidity", m)
	humiditySensor.Manufacturer = "väder"
	humiditySensor.Name = "Relative Humidity (outside)"
	humiditySensor.LastWillID = id
	humiditySensor.Type = "humiditySensor"
	humiditySensor.AddFeature("currentRelativeHumidity", &device.Feature{})
	h.devices = append(h.devices, humiditySensor)

	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("Failed to establish connection with broker: ", token.Error())
	}
	// Publish the first time
	do(*apiToken, *location, *refresh, tempSensor, humiditySensor)

loop:
	for {
		select {
		case sig := <-quit:
			log.Printf("Received signal: %s, proceeding to shutdown", sig)
			break loop
		// Publish after every interval has elapsed
		case <-time.After(time.Duration(*refresh) * time.Hour):
			do(*apiToken, *location, *refresh, tempSensor, humiditySensor)
		}
	}

	c.Disconnect(250)
	log.Print("Disconnected from broker. Bye!")
	os.Exit(0)
}

// do executes a fetch and publish cycle
func do(token string, location string, interval int, sensors ...*device.Device) {
	conditions, err := vader.GetWeather(token, location)
	if err != nil {
		log.Printf("Failed to get weather: %s. Next attempt in %d hours", err, interval)
		return
	}

	for _, sensor := range sensors {
		switch sensor.Type {
		case "temperatureSensor":
			ft, _ := sensor.GetFeature("currentTemperature")
			ft.Update(strconv.FormatFloat(float64(conditions.FeelsLikeC), 'E', -1, 32))
			log.Print("Published current temperature")
		case "humiditySensor":
			ft, _ := sensor.GetFeature("currentRelativeHumidity")
			ft.Update(strings.Trim(conditions.RelativeHumidity, "%"))
			log.Print("Published current relative humidity")
		}
	}
}

func (h *handler) onConnectHandler(c mq.Client) {
	log.Print("Connected to MQTT broker")

	c.Subscribe("discover", 1, func(mq.Client, mq.Message) {
		for _, dev := range h.devices {
			dev.PublishMeta()
		}
	})
}
