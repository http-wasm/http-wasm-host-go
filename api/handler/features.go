package handler

import (
	"strings"
)

// Features is a bit flag of features a host may support.
//
// Note: Numeric values are not intended to be interpreted except as bit flags.
type Features uint64

const (
	// FeatureBufferRequest buffers the HTTP request body when reading, so that
	// FuncNext can see the original.
	//
	// Note: Buffering a request is done on the host and can use resources
	// such as memory. It also may reduce the features of the underlying
	// request due to implications of buffering or wrapping.
	FeatureBufferRequest Features = 1 << iota

	// FeatureBufferResponse buffers the HTTP response produced by FuncNext
	// instead of sending it immediately. This allows the caller to inspect and
	// overwrite the HTTP status code or response body. As the response is
	// deferred, may experience timing differences with this enabled.
	//
	// Note: Buffering a response is done on the host and can use resources
	// such as memory. It also may reduce the features of the underlying
	// response due to implications of buffering or wrapping.
	FeatureBufferResponse
)

// WithEnabled enables the feature or group of features.
func (f Features) WithEnabled(feature Features) Features {
	return f | feature
}

// IsEnabled returns true if the feature (or group of features) is enabled.
func (f Features) IsEnabled(feature Features) bool {
	return f&feature != 0
}

// String implements fmt.Stringer by returning each enabled feature.
func (f Features) String() string {
	var builder strings.Builder
	for i := 0; i <= 63; i++ { // cycle through all bits to reduce code and maintenance
		target := Features(1 << i)
		if f.IsEnabled(target) {
			if name := featureName(target); name != "" {
				if builder.Len() > 0 {
					builder.WriteByte('|')
				}
				builder.WriteString(name)
			}
		}
	}
	return builder.String()
}

func featureName(f Features) string {
	switch f {
	case FeatureBufferRequest:
		return "buffer-request"
	case FeatureBufferResponse:
		return "buffer-response"
	}
	return ""
}
