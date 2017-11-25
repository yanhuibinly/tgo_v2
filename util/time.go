package util

import (
	"fmt"
	"time"
)

func TimeGetDate(t time.Time) time.Time {

	year, month, day := t.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func TimeIsToday(t time.Time) bool {
	return TimeSameDay(t, time.Now())
}

func TimeSameDay(t1 time.Time, t2 time.Time) bool {
	if TimeDiffDay(t1, t2) == 0 {
		return true
	}
	return false
}

func TimeDiffDay(t1 time.Time, t2 time.Time) int {
	return int(TimeGetDate(t2).Sub(TimeGetDate(t1)) / (24 * time.Hour))
}

func TimeFewDaysLater(day int) time.Time {
	return TimeFewDurationLater(time.Duration(day) * 24 * time.Hour)
}

func TimeTwentyFourHoursLater() time.Time {
	return TimeFewDurationLater(time.Duration(24) * time.Hour)
}

func TimeSixHoursLater() time.Time {
	return TimeFewDurationLater(time.Duration(6) * time.Hour)
}

func TimeFewDurationLater(duration time.Duration) time.Time {
	baseTime := time.Now()
	fewDurationLater := baseTime.Add(duration)
	return fewDurationLater
}

func TimeIsExpired(expirationTime time.Time) bool {
	after := time.Now().After(expirationTime)
	return after
}

//TimeConsumePrint print time consume
func TimeConsumePrint(timeBefore time.Time, prefix string) {
	duration := time.Now().Sub(timeBefore)
	fmt.Printf("%s time :%d\n", prefix, duration.Nanoseconds()/1000000)
}

func TimeIsLeapYear(y int) bool {
	if (y%400 == 0) || (y%4 == 0 && y%100 != 0) {
		return true
	} else {
		return false
	}
}
