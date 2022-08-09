package logger

var globalLogger = NewNoop()

func SetGlobal(l *Logger) {
	globalLogger = l
}

func L() *Logger {
	return globalLogger
}
