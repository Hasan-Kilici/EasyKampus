package handlers

// ErrorDetail carries the machine readable code and the human readable
// message of a failed request.
//
//	@name	ErrorDetail
type ErrorDetail struct {
	Code    string `json:"code" example:"NOT_FOUND"`
	Message string `json:"message" example:"record not found"`
}

// ErrorResponse is the payload returned by every failing endpoint. Successful
// endpoints use the same envelope with a null error object.
//
//	@name	ErrorResponse
type ErrorResponse struct {
	Data  interface{} `json:"data"`
	Error ErrorDetail `json:"error"`
}
