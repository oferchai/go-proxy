package utils

import (
	"fmt"
	"time"
)

func GetTimeFrame(t time.Time) string {
	hour := t.Hour()
	timeFrame := hour - (hour % 3)
	return fmt.Sprintf("%02d-%02d", timeFrame, timeFrame+3)
}

func GenerateIPKey(ip string, t time.Time) string {
	dateStr := t.Format("2006-01-02")
	timeFrame := GetTimeFrame(t)
	return fmt.Sprintf("IP:%s-%s-%s", ip, dateStr, timeFrame)
}
