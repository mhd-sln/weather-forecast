package main

import (
	"encoding/json"
	"flag"
	"github.com/jasonlvhit/gocron"
	"github.com/salmanmanekia/services"
	"io/ioutil"
	"net/http"
	"strconv"
)

const (
	APIKey      = "9ccb28771f140ffbc82149932fe43172"
	ExamplePath = "/path/to/the/config/file"
	DefaultPath = "config.json"
)

type WeatherData struct {
	City struct {
		Name string `json:name`
	} `json:city`
	Temps []struct {
		Date uint32 `json:"dt"`
		Main struct {
			Temp float32 `json:temp`
		} `json:"main"`
	} `json:"list"`
}

type ForecastConfig struct {
	AtLeastTemp      int    `json:"not_less_than_temperature"`
	AtMostTemp       int    `json:"not_more_than_temperature"`
	CityIds          []int  `json:"city"`
	CheckingInterval uint64 `json:"checking_interval"`
}

func DefaultConfig() *ForecastConfig {
	return &ForecastConfig{
		AtLeastTemp:      -15,
		AtMostTemp:       30,
		CityIds:          []int{1174872, 5106292, 658224},
		CheckingInterval: 3,
	}
}

type weatherData WeatherData

var weatherForecast []*WeatherData

func printForecast() {
	for _, wf := range weatherForecast {
		out, _ := json.Marshal(wf)
		logger.Info("%q", string(out))
	}
}

func (wd *WeatherData) UnmarshallJSON(b []byte) (err error) {
	data := weatherData{}
	if err = json.Unmarshal(b, &data); err == nil {
		*wd = WeatherData(data)
		return
	}
	return
}

func requests(cities []int) {
	ch := make(chan []byte)

	for _, city := range cities {
		go getter(city, ch)
	}

	for i := 0; i < len(cities); i++ {
		var wd WeatherData
		wd.UnmarshallJSON(<-ch)

		weatherForecast = append(weatherForecast, &wd)
		out, err := json.Marshal(wd)
		if err != nil {
			logger.Error("%q", err)
		}
		logger.Info("%s", string(out))
	}
}

func getter(city int, ch chan []byte) {
	respBody, err := get(city)
	if err == nil {
		ch <- respBody
	} else {
		logger.Error("%q", err)
	}

}

func get(city int) ([]byte, error) {
	url := "http://api.openweathermap.org/data/2.5/forecast?APPID=" + APIKey + "&id=" + strconv.Itoa(city) + "&units=metric"

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func main() {

	var forecastConfig *ForecastConfig
	var config string
	flag.StringVar(&config, "config", ExamplePath, "a string")
	flag.Parse()

	if config == ExamplePath {
		config = DefaultPath
	}

	contents, err := ioutil.ReadFile(config)
	if err == nil {
		err = json.Unmarshal(contents, &forecastConfig)
	}

	if err != nil {
		logger.Warn("%q", err)
		forecastConfig = DefaultConfig()
	}

	cities := forecastConfig.CityIds

	s := gocron.NewScheduler()
	s.Every(forecastConfig.CheckingInterval).Seconds().Do(requests, cities)

	<-s.Start()
}
