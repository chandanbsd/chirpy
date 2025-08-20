package helper

import "net/http"

func ReportError(message string, resWriter http.ResponseWriter, httpResCode int) {
	resWriter.WriteHeader(httpResCode)
	resWriter.Write([]byte(message))
}

func RespondSuccess(resWriter http.ResponseWriter, httpResCode int, resBodyBytes []byte) {
	resWriter.WriteHeader(200)
	resWriter.Write(resBodyBytes)
}
