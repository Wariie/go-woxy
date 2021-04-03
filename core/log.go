package core

import "github.com/sirupsen/logrus"

func (core *Core) GetLogger(name string) *logrus.Logger {
	return core.loggers[name]
}
