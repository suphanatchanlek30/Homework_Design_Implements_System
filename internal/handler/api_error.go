package handler

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// newErrorResponse builds the shared API error payload used by handlers.
// สร้าง payload error มาตรฐานที่ handler ทุกตัวใช้ร่วมกัน
func newErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}
