package helpers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
)

func ConvertToInt(str string) int {

	tempStr, errConv := strconv.Atoi(str)
	if errConv != nil {
		fmt.Println(errConv)
	}
	return tempStr
}

func ConvertToString(str int) string {

	tempStr := strconv.Itoa(str)
	return tempStr
}
func ConvertToString64(str int64) string {

	tempStr := strconv.FormatInt(str, 10)
	return tempStr
}

func ConvertFloatToString(v float64) string {

	s64 := strconv.FormatFloat(v, 'f', -10, 32)
	return s64
}

func ConvertToFloat(str string) float64 {

	tempStr, errConv := strconv.ParseFloat(str, 32)
	if errConv != nil {
		fmt.Println(errConv)
	}
	return RoundFloat(tempStr, 2)
}

func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
func ParseHttpRequest(r *http.Request) {

	log.Println("Method: " + r.Method + "  Query: " + r.URL.Path)
	headerValue := r.Header.Get("Bearer")
	if len(headerValue) > 0 {
		log.Println("Bearer: ", headerValue)
	}

}
