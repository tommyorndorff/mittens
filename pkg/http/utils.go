//Copyright 2019 Expedia, Inc.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package http

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Request represents an HTTP request.
type Request struct {
	Method string
	Path   string
	Body   *string
}

var allowedHTTPMethods = map[string]interface{}{
	"GET":     nil,
	"HEAD":    nil,
	"POST":    nil,
	"PUT":     nil,
	"PATCH":   nil,
	"DELETE":  nil,
	"CONNECT": nil,
	"OPTIONS": nil,
	"TRACE":   nil,
}

// anything that starts with {$, followed by any word character, and optionally followed by a modifier identifier | and the modifiers that can contain word chars + - = and ,
var templatePlaceholderRegex = regexp.MustCompile("{\\$(\\w+(?:[\\|(?:[\\w+-=,]+)]*)}")
var templateRangeRegex = regexp.MustCompile("{\\$range\\|min=(?P<Min>\\d+),max=(?P<Max>\\d+)}")
var templateElementsRegex = regexp.MustCompile("{\\$random\\|(?P<Elements>[,\\w-]+)}")
var templateDatesRegex = regexp.MustCompile("{\\$currentDate(?:\\|(?:days(?P<Days>[+-]\\d+))*(?:[,]*months(?P<Months>[+-]\\d+))*(?:[,]*years(?P<Years>[+-]\\d+))*)*}")

// ToHTTPRequest parses an HTTP request which is in a string format and stores it in a struct.
func ToHTTPRequest(requestString string) (Request, error) {
	parts := strings.SplitN(requestString, ":", 3)
	if len(parts) < 2 {
		return Request{}, fmt.Errorf("invalid request flag: %s, expected format <http-method>:<path>[:body]", requestString)
	}

	method := strings.ToUpper(parts[0])
	_, ok := allowedHTTPMethods[method]
	if !ok {
		return Request{}, fmt.Errorf("invalid request flag: %s, method %s is not supported", requestString, method)
	}

	// <method>:<path>
	if len(parts) == 2 {
		path := interpolatePlaceholders(parts[1])

		return Request{
			Method: method,
			Path:   path,
			Body:   nil,
		}, nil
	}

	path := interpolatePlaceholders(parts[1])
	var body = interpolatePlaceholders(parts[2])

	return Request{
		Method: method,
		Path:   path,
		Body:   &body,
	}, nil
}

// dateElements replaces date placeholders with the actual dates. It supports offsets for days, months, and years.
func dateElements(source string) string {
	r := templateDatesRegex.FindStringSubmatch(source)

	if r == nil {
		return source
	}
	days := r[1]
	months := r[2]
	years := r[3]

	offsetDays, _ := strconv.Atoi(days)
	offsetMonths, _ := strconv.Atoi(months)
	offsetYears, _ := strconv.Atoi(years)

	// the date below is how the golang date formatter works. it's used for the formatting. it's not what is actually going to be displayed
	return time.Now().AddDate(offsetYears, offsetMonths, offsetDays).Format("2006-01-02")
}

// timestampElements returns the current time from Unix epoch in milliseconds.
func timestampElements() string {
	epoch := time.Now().UnixNano() / 1000000

	return strconv.FormatInt(epoch, 10)
}

// randomElements replaces random element placeholders with elements which are randomly selected from the provided list.
func randomElements(source string) string {
	r := templateElementsRegex.FindStringSubmatch(source)

	if r == nil {
		return source
	}

	s := strings.Split(r[1], ",")
	number := rand.Intn(len(s))

	return s[number]
}

// rangeElements replaces range element placeholders with random integers within the specified range.
func rangeElements(source string) string {
	r := templateRangeRegex.FindStringSubmatch(source)
	if r == nil {
		return source
	}

	min, _ := strconv.Atoi(r[1])
	max, _ := strconv.Atoi(r[2])

	if min > max {
		log.Printf("Invalid range. min > max")
		return source
	}

	number := rand.Intn(max-min+1) + min

	return strconv.Itoa(number)
}

// interpolatePlaceholders scans a string and replaces placeholders with actual values.
// At the moment this supports; dates, timestamps, random values from a list, and random integers.
func interpolatePlaceholders(source string) string {
	return templatePlaceholderRegex.ReplaceAllStringFunc(source, func(templateString string) string {

		if strings.Contains(templateString, "currentDate") {
			return dateElements(templateString)
		} else if strings.Contains(templateString, "currentTimestamp") {
			return timestampElements()
		} else if strings.Contains(templateString, "random") {
			return randomElements(templateString)
		} else if strings.Contains(templateString, "range") {
			return rangeElements(templateString)
		} else {
			return source
		}
	})
}
