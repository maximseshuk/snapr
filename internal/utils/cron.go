package utils

import (
	"time"

	"github.com/robfig/cron/v3"
)

func GetNextRunTime(cronExpr string) (time.Time, error) {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	return schedule.Next(time.Now()), nil
}
