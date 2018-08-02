package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hemtjanst/bibliotek/client"
	"github.com/hemtjanst/bibliotek/device"
	"github.com/hemtjanst/bibliotek/feature"
	"github.com/hemtjanst/bibliotek/transport/mqtt"
	"github.com/hemtjanst/vader"
)

var (
	location = flag.String("location", "autoip", "Location to fetch the current conditions of")
	refresh  = flag.Int("refresh", 1, "Time in hours after which to query the Wunderground API for new data")
	apiToken = flag.String("token", "REQUIRED", "Wunderground API token")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Parameters:\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}
	mcfg := mqtt.MustFlags(flag.String, flag.Bool)
	flag.Parse()

	if *apiToken == "REQUIRED" {
		log.Fatal("A token is required to be able to query the Wunderground API\n")
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	m, err := mqtt.New(ctx, mcfg())
	if err != nil {
		panic(err)
	}

	tempSensor, _ := client.NewDevice(&device.Info{
		Topic:        "sensor/temperature/wunderground",
		Manufacturer: "väder",
		Name:         "Temperature (outside)",
		Type:         "temperatureSensor",
		Features: map[string]*feature.Info{
			"currentTemperature": &feature.Info{
				Min: -50,
			}},
	}, m)
	humiditySensor, _ := client.NewDevice(&device.Info{
		Topic:        "sensor/humidity/wunderground",
		Manufacturer: "vader",
		Name:         "Relative Humidity (outside)",
		Type:         "humiditySensor",
		Features: map[string]*feature.Info{
			"currentRelativeHumidity": &feature.Info{}},
	}, m)

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

	cancel()
	log.Print("Disconnected from broker. Bye!")
	os.Exit(0)
}

// do executes a fetch and publish cycle
func do(token string, location string, interval int, sensors ...client.Device) {
	conditions, err := vader.GetWeather(token, location)
	if err != nil {
		log.Printf("Failed to get weather: %s. Next attempt in %d hours", err, interval)
		return
	}

	for _, sensor := range sensors {
		switch sensor.Type() {
		case "temperatureSensor":
			ft := sensor.Feature("currentTemperature")
			ft.Update(strconv.FormatFloat(float64(conditions.FeelsLikeC), 'f', 1, 32))
			log.Print("Published current temperature")
		case "humiditySensor":
			ft := sensor.Feature("currentRelativeHumidity")
			ft.Update(strings.Trim(conditions.RelativeHumidity, "%"))
			log.Print("Published current relative humidity")
		}
	}
}
