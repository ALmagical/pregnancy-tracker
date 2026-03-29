package timeutil

import "time"

var shanghai *time.Location

func init() {
	var err error
	shanghai, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		shanghai = time.FixedZone("CST", 8*3600)
	}
}

func Shanghai() *time.Location { return shanghai }

func TodayDate() time.Time {
	now := time.Now().In(Shanghai())
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, Shanghai())
}

func ParseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, Shanghai())
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func GestationalDays(lastPeriod time.Time) int {
	today := TodayDate()
	lp := lastPeriod.In(Shanghai())
	y, m, d := lp.Date()
	lp0 := time.Date(y, m, d, 0, 0, 0, 0, Shanghai())
	days := int(today.Sub(lp0).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func DueFromLMP(lastPeriod time.Time) time.Time {
	lp := lastPeriod.In(Shanghai())
	return lp.AddDate(0, 0, 280)
}

func WeekDayFromGestationalDays(days int) (week int, day int) {
	if days < 0 {
		return 0, 0
	}
	return days / 7, days % 7
}
