package jalali

import (
	"fmt"
	"strings"
	"time"

	persian "github.com/yaa110/go-persian-calendar"
)

func ParseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid jalali date %q", value)
	}
	var y, m, d int
	if _, err := fmt.Sscanf(parts[0], "%d", &y); err != nil {
		return time.Time{}, err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil {
		return time.Time{}, err
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &d); err != nil {
		return time.Time{}, err
	}
	p := persian.Date(y, persian.Month(m), d, 0, 0, 0, 0, time.UTC)
	return p.Time(), nil
}

func CalendarDaysUntil(from, to time.Time) int {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(to.Sub(from).Hours() / 24)
}
