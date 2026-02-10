package utils

type IpAddressError struct{}
type UnknownProviderError struct{}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (e IpAddressError) Error() string {
	return "invalid IP address"
}

func (e UnknownProviderError) Error() string {
	return "unknown provider"
}
