package recaptcha

// Config is a struct for specifying configuration options for the recaptcha middleware.
type Config struct {
	// Extractor is the method for extracting recaptcha token from request
	// Default: FromHeader (i.e., from recaptcha header as token)
	Extractor TokenExtractor
	// The function that will be called when there's an error validating the token
	// Default value:
	ErrorHandler errorHandler
}
