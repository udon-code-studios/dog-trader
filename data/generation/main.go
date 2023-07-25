/*
DataGeneration creates files with stock bar data and technical indicators.

This program uses relative file paths. It is intended to be run from the
directory: dog-trader/data/generation

Output file names will be in the format: ../tickers/SYMBOL/SYMBOL_YYYYMMDD_vXXX.csv
(vXXX indicates the output file version number). File versions are used to
track column content, data format, and indicator calculation metholodogy
changes. Version change information is tracked in the README file.

Output files will contain data from 8:30 AM to 4:00PM EST.

Current output file version: v000

Usage:

	generation [flags]

The flags are:

	-t SYMBOL1,SYMBOL2
		comma-separated ticker symbol list (required)

	-s YYYYMMDD
		start date (inclusive) (required)

	-e YYYYMMDD
		end date (inclusive) (default: today)
*/
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
)

// define constants
const OUTPUT_VERSION = "v000"

func main() {
	//-------------------------------------------------------------------------
	// Ensure Alpaca environment variables are set
	//-------------------------------------------------------------------------

	checkAlpacaEnvVars()

	//-------------------------------------------------------------------------
	// Parse flags for start, end, and tickers
	//-------------------------------------------------------------------------

	start, end, tickers := parseFlags()

	// print parsed flags
	fmt.Println("[ INFO ] Start date:", start)
	fmt.Println("[ INFO ] End date:", end)
	fmt.Println("[ INFO ] Tickers:", tickers)

	//-------------------------------------------------------------------------
	// data generation loop
	//-------------------------------------------------------------------------

	holidays := getHolidays("../../properties/market-holidays.json")

	newYork, _ := time.LoadLocation("America/New_York")

	fmt.Println("[ INFO ] --------------------------------------------------")
	fmt.Println("[ INFO ] Stating data generation loop...")

	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		// skip weekends and holidays (including early closes)
		if isWeekend(current) || isHoliday(current, holidays) {
			continue
		}

		startTime := time.Date(current.Year(), current.Month(), current.Day(), 8, 0, 0, 0, newYork) // 8:00 AM
		endTime := time.Date(current.Year(), current.Month(), current.Day(), 16, 0, 0, 0, newYork)  // 4:00 PM

		for _, symbol := range tickers {
			// define output directory and filename
			outputDirectory := fmt.Sprintf("../tickers/%s", symbol)
			outputFilename := fmt.Sprintf("%s_%d%02d%02d_%s.csv", symbol, current.Year(), current.Month(), current.Day(), OUTPUT_VERSION)

			fmt.Println("[ INFO ] Generating file:", outputFilename)

			// get minute bars for day and ticker from Alpaca
			bars, err := marketdata.GetBars(symbol, marketdata.GetBarsRequest{
				TimeFrame: marketdata.OneMin,
				Start:     startTime,
				End:       endTime,
			})
			panicOnNil(err)

			// fill in bars that do not exist
			for i := 0; i < len(bars); i++ {
				if i == 0 {
					// if first minute is not 8:00 AM, then exit
					if !bars[i].Timestamp.Equal(startTime) {
						fmt.Println("[ ERROR ] First minute bar is not 8:00 AM")
						os.Exit(1)
					}

					// skip 8:00 AM minute
					continue
				}

				// check for non-sequantial minute bars
				if !bars[i].Timestamp.Equal(bars[i-1].Timestamp.Add(time.Minute)) {
					bars = insertBar(bars, i, marketdata.Bar{
						Timestamp:  bars[i-1].Timestamp.Add(time.Minute),
						Open:       bars[i-1].Close,
						High:       bars[i-1].Close,
						Low:        bars[i-1].Close,
						Close:      bars[i-1].Close,
						Volume:     0,
						TradeCount: 0,
						VWAP:       0,
					})
				}
			}

			// create output file and writer
			err = os.MkdirAll(outputDirectory, 0755)
			panicOnNil(err)
			file, err := os.Create(fmt.Sprintf("%s/%s", outputDirectory, outputFilename))
			panicOnNil(err)
			writer := csv.NewWriter(file)
			defer writer.Flush()

			// write header
			header := []string{
				"time",
				"open",
				"high",
				"low",
				"close",
				"volume",
			}
			writer.Write(header)

			// create and write rows
			for _, bar := range bars {
				// skip rows prior to 8:30 AM
				if bar.Timestamp.Before(time.Date(current.Year(), current.Month(), current.Day(), 8, 30, 0, 0, newYork)) {
					continue
				}

				// get time
				time := fmt.Sprintf("%02d:%02d", bar.Timestamp.In(newYork).Hour(), bar.Timestamp.In(newYork).Minute())

				// get OHLC
				open := fmt.Sprintf("%.3f", bar.Open)
				high := fmt.Sprintf("%.3f", bar.High)
				low := fmt.Sprintf("%.3f", bar.Low)
				close := fmt.Sprintf("%.3f", bar.Close)

				// get volume
				volume := fmt.Sprintf("%d", bar.Volume)

				// write row
				writer.Write([]string{time, open, high, low, close, volume})
			}
		}
	}
}

//----------------------------------------------------------------------------
// helper functions
//----------------------------------------------------------------------------

func panicOnNil(value interface{}) {
	if value != nil {
		panic(value)
	}
}

/*
Checks if all Alpaca environment variables are set, exiting with status 1 if
not.
*/
func checkAlpacaEnvVars() {
	alpacaEnvVars := []string{"APCA_API_KEY_ID", "APCA_API_SECRET_KEY"}

	for _, envVar := range alpacaEnvVars {
		if os.Getenv(envVar) == "" {
			fmt.Println("[ ERROR ] The environment variable", envVar, "is not set")
			os.Exit(1)
		}
	}
}

/*
Parses the command line arguments for start and end dates and the ticker
symbol list, exiting with status 1 if there is a missing required flag or an
error is encountered while parsing.
*/
func parseFlags() (start, end time.Time, tickers []string) {
	// define and parse flags
	startFlag := flag.String("s", "", "Start date (inclusive) (format: YYYYMMDD) (required)")
	endFlag := flag.String("e", "", "End date (inclusive) (format: YYYYMMDD) (default: today)")
	tickersFlag := flag.String("t", "", "Comma-separated list of ticker symbols (format: SYMBOL1,SYMBOL2) (required)")
	flag.Parse()

	// require start flag to be specified
	if *startFlag == "" {
		fmt.Println("[ ERROR ] Start date flag (-s) is missing")
		os.Exit(1)
	}

	// require ticker symbol list to be specified
	if *tickersFlag == "" {
		fmt.Println("[ ERROR ] Ticker symbol list flag (-t) is missing")
		os.Exit(1)
	}

	// get the start and end dates as time.Time objects
	start, err := time.Parse("20060102", *startFlag)
	if err != nil {
		fmt.Println("[ ERROR ] Error while parsing start date:", err)
		os.Exit(1)
	}
	end, err = time.Parse("20060102", *endFlag)
	if err != nil {
		fmt.Println("[ ERROR ] Error while parsing end date:", err)
		os.Exit(1)
	}

	// get the ticker symbols as a slice
	tickers = strings.Split(*tickersFlag, ",")

	return
}

func getHolidays(filepath string) (holidays map[string][]struct {
	Month      int
	Day        int
	EarlyClose bool
}) {
	// Open the JSON file.
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("[ ERROR ] Error opening ../../properties/market-holidays.json:", err)
		os.Exit(1)
	}

	// Read the JSON data from the file.
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("[ ERROR ] Error reading file:", err)
		os.Exit(1)
	}

	// Unmarshal the JSON data into a map.
	err = json.Unmarshal(data, &holidays)
	if err != nil {
		fmt.Println("[ ERROR ] Error unmarshalling JSON:", err)
		os.Exit(1)
	}

	return holidays
}

func isHoliday(date time.Time, holidays map[string][]struct {
	Month      int
	Day        int
	EarlyClose bool
}) bool {
	// Check if the date is a holiday.
	isHoliday := false
	for _, holiday := range holidays[strconv.Itoa(date.Year())] {
		if holiday.Month == int(date.Month()) && holiday.Day == date.Day() {
			isHoliday = true
			break
		}
	}

	return isHoliday
}

func isWeekend(date time.Time) bool {
	return date.Weekday() == 0 || date.Weekday() == 6
}

func insertBar(bars []marketdata.Bar, index int, bar marketdata.Bar) []marketdata.Bar {
	// Create a new array.
	newArray := make([]marketdata.Bar, len(bars)+1)

	// Copy the elements from the old array to the new array.
	for i := 0; i < index; i++ {
		newArray[i] = bars[i]
	}

	// Insert the new element into the new array.
	newArray[index] = bar

	// Copy the remaining elements from the old array to the new array.
	for i := index + 1; i < len(bars)+1; i++ {
		newArray[i] = bars[i-1]
	}

	// Return the new array.
	return newArray
}
