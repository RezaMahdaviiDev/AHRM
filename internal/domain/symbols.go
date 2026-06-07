package domain

const (
	UnderlyingSymbol = "اهرم"
	CallOptionPrefix = "ضهرم"
	PutOptionPrefix  = "طهرم"
)

func IsCallOption(symbol string) bool {
	return hasPrefix(symbol, CallOptionPrefix)
}

func IsPutOption(symbol string) bool {
	return hasPrefix(symbol, PutOptionPrefix)
}

func IsAHRMUnderlying(symbol string) bool {
	return symbol == UnderlyingSymbol
}

func hasPrefix(symbol, prefix string) bool {
	if len(symbol) < len(prefix) {
		return false
	}
	return symbol[:len(prefix)] == prefix
}
