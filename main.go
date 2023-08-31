package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	failedLogins = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "failed_logins_total",
		Help: "The total number of failed login attempts",
	}, []string{
		"ip",
		"username",
		"country",
		"city",
		"continent",
		"latitude",
		"longitude",
		"postal",
		"isp",
	})
)

func init() {
	// Register the Prometheus HTTP handler
	http.Handle("/metrics", promhttp.Handler())
}

func main() {
	err := godotenv.Load("./ivan.env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}
	token := os.Getenv("TOKEN")
	// Specify the file path you want to monitor
	filePath := "./auth.log"

	// Create a new watcher instance
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add the file to the watcher
	err = watcher.Add(filePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Monitoring changes to %s...\n", filePath)
	go func() {
		log.Fatal(http.ListenAndServe(":9101", nil))
	}()
	// Start a loop to watch for events
	for {
		select {
		case event := <-watcher.Events:
			// Check if the event is a modification
			fmt.Println("Event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				readLastLine(filePath, token)
			}
		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}

func readLastLine(filePath string, token string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	if strings.Contains(lastLine, "Failed password for") {
		username, ip := extractIP(lastLine)
		geoData := getGeoData(ip, token)
		// printf statement to print username ip and Country name
		fmt.Printf("Username: %s, IP: %s, Country: %s\n", username, ip, geoData.Country.Name["en"])
		recordMetrics(geoData, username, ip) // Record metrics for Prometheus

	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
		return
	}
}

func extractIP(line string) (string, string) {
	parts := strings.Split(line, " ")
	if strings.Contains(line, "Failed password for invalid user") {
		return parts[10], parts[12]
	}
	return parts[8], parts[10]
}

func getGeoData(ip string, token string) GeoData {
	url := "https://api.findip.net/" + ip + "/?token=" + token
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return GeoData{}
	}
	defer response.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return GeoData{}
	}

	var geoData GeoData

	err = json.Unmarshal(responseBody, &geoData)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return GeoData{}
	}

	return geoData
}

func recordMetrics(geoData GeoData, username string, ip string) {
	failedLogins.With(prometheus.Labels{
		"ip":        ip,
		"username":  username,
		"country":   geoData.Country.Name["en"],
		"city":      geoData.City.Names["en"],
		"continent": geoData.Continent.Code,
		"latitude":  fmt.Sprintf("%.6f", geoData.Location.Latitude),
		"longitude": fmt.Sprintf("%.6f", geoData.Location.Longitude),
		"postal":    geoData.Postal.Code,
		"isp":       geoData.Traits.ISP,
	}).Inc()
}

type GeoData struct {
	Country   Country   `json:"country"`
	City      City      `json:"city"`
	Continent Continent `json:"continent"`
	Location  Location  `json:"location"`
	Postal    Postal    `json:"postal"`
	Traits    Traits    `json:"traits"`
}

type Country struct {
	Name              map[string]string `json:"names"`
	IsInEuropeanUnion bool              `json:"is_in_european_union"`
	ISOCode           string            `json:"iso_code"`
}

type City struct {
	Names map[string]string `json:"names"`
}

type Continent struct {
	Code  string            `json:"code"`
	Names map[string]string `json:"names"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	TimeZone  string  `json:"time_zone"`
}

type Postal struct {
	Code string `json:"code"`
}

type Traits struct {
	ISP string `json:"isp"`
}
