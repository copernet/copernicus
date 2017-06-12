package log

type LogClosure func() string

func (c LogClosure) ToString() string {
	return c()
}
func InitLogClosure(c func() string) LogClosure {
	return LogClosure(c)
}
