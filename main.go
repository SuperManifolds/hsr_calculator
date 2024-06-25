package main

import (
	"image/color"
	"log"
	"math"
	"os"
	"sort"

	"github.com/gocarina/gocsv"
	"github.com/twpayne/go-kml"
)

type City struct {
    Area       string  `csv:"Areas"`
    Latitude   float64 `csv:"Latitude"`
    Longitude  float64 `csv:"Longtitude"`
    Population int64   `csv:"Population"`
}

type CityPair struct {
    City1 string `csv:"City1"`
    City2 string `csv:"City2"`
    Distance float64 `csv:"Distance"`
    FlightTime float64 `csv:"FlightTime"`
    DrivingTime float64 `csv:"DrivingTime"`
    Gravity float64 `csv:"Gravity"`
    Score float64 `csv:"Score"`
}

func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Radius of the Earth in kilometers. Use 3959 for miles.

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c

	return distance
}

func GravityModel(pop1, pop2, time float64) float64 {
    return ((pop1/1000) * (pop2/1000)) / (time * time)
}

func FlightTime(dist float64) float64 {
    return (dist*0.068)+55+200
}

func DrivingTime(dist float64) float64 {
    return dist+30
}

func TrainTime(dist float64) float64 {
    return (0.22718*dist)+25+40
}

func scoreOpacity(value int) uint8 {
    if value >= 9000 {
		return 255
	} else if value <= 0 {
		return 128
	}
	// Scale proportionally between 128 and 255
	return uint8(float64(value)/9000*127) + 128
}


func HSRCoefficient(x float64) float64 {
	if x <= 50 {
		return 0
	} else if x <= 400 {
		return (x - 50) / 350
	} else if x < 1200 {
		return 1 - (x - 400) / 800
	} else {
		return 0
	}
}

func main () {
    file, err := os.Open("cities.csv")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    var cities []City
    if err := gocsv.UnmarshalFile(file, &cities); err != nil {
        log.Fatal(err)
    }

    var cityPairs []CityPair
    var cityPairMap = make(map[string]bool)
    placemarks := make([]kml.Element, 0)
    for _, city1 := range cities {
        println(city1.Area)
        cityPlacemark := kml.Placemark(
            kml.Name(city1.Area),
            kml.Point(
                kml.Coordinates(kml.Coordinate{Lon: city1.Longitude, Lat: city1.Latitude}),
            ),
        )
        placemarks = append(placemarks, cityPlacemark)

        for _, city2 := range cities {
            if cityPairMap[city1.Area + city2.Area] || cityPairMap[city2.Area + city1.Area] {
                continue
            }
            dist := Haversine(city1.Latitude, city1.Longitude, city2.Latitude, city2.Longitude)

            flightTime := FlightTime(dist)
            drivingTime := DrivingTime(dist)
            trainTime := TrainTime(dist)
            gravity := GravityModel(float64(city1.Population), float64(city2.Population), trainTime)
            score := gravity * HSRCoefficient(dist)

            //println(city1.Area, city2.Area, dist, flightTime, drivingTime, gravity, score)
            if score < 100 {
                continue
            }

            cityPair := CityPair{
                City1: city1.Area,
                City2: city2.Area,
                Distance: dist,
                FlightTime: flightTime,
                DrivingTime: drivingTime,
                Gravity: gravity,
                Score: score,
            }
            cityPairs = append(cityPairs, cityPair)
            cityPairMap[city1.Area + city2.Area] = true

            lineWidth := math.Floor(score / 100)

            line := kml.LineString(
			kml.Coordinates(kml.Coordinate{Lon: city1.Longitude, Lat: city1.Latitude}, kml.Coordinate{Lon: city2.Longitude, Lat: city2.Latitude},),
		    )

            style := kml.Style(
				kml.LineStyle(
					kml.Color(color.RGBA{R: 255, G: 255, B: 255, A: scoreOpacity(int(score))}),
					kml.Width(lineWidth),
				),
			)
		    placemarks = append(placemarks, kml.Placemark(style, line))
        }
    }

    sort.Slice(cityPairs, func(i, j int) bool {
		return cityPairs[i].Score > cityPairs[j].Score
	})

    file, err = os.Create("city_pairs.csv")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    err = gocsv.MarshalFile(&cityPairs, file)
    if err != nil {
        log.Fatal(err)
    }
    doc := kml.Document(placemarks...)
    lineFile, err := os.Create("lines.kml")
	if err != nil {
		panic(err)
	}
	defer lineFile.Close()

	if err := kml.KML(doc).Write(lineFile); err != nil {
		panic(err)
	}
}
