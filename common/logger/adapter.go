package logger

// Adapter can be used as an adapter for logging from other frameworks/libraries.
// Just keep adding the required methods to make it function.
type Adapter Logger

func (log *Adapter) Log(msg string) {
	if log == nil {
		return
	}
	(*Logger)(log).Info(msg)
}
