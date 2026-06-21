package server

import (
	"fmt"
	"time"
)

// toJalali converts a YYYY-MM-DD string to Jalali (Persian) YYYY/MM/DD.
func toJalali(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	jy, jm, jd := gregorianToJalali(t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%d/%02d/%02d", jy, jm, jd)
}

// gregorianToJalali converts a Gregorian date to Jalali (Solar Hijri).
// Verified: 2026-06-21 → 1405/03/31.
func gregorianToJalali(gy, gm, gd int) (int, int, int) {
	gDays := [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	jDays := [12]int{31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30, 29}

	gY := gy - 1600
	gM := gm - 1
	gD := gd - 1

	gDayNo := 365*gY + (gY+3)/4 - (gY+99)/100 + (gY+399)/400
	for i := 0; i < gM; i++ {
		gDayNo += gDays[i]
	}
	if gM > 1 && ((gy%4 == 0 && gy%100 != 0) || gy%400 == 0) {
		gDayNo++
	}
	gDayNo += gD

	jDayNo := gDayNo - 79
	jNp := jDayNo / 12053
	jDayNo %= 12053

	jY := 979 + 33*jNp + 4*(jDayNo/1461)
	jDayNo %= 1461

	if jDayNo >= 366 {
		jY += (jDayNo - 1) / 365
		jDayNo = (jDayNo - 1) % 365
	}

	jM := 0
	for i := 0; i < 11 && jDayNo >= jDays[i]; i++ {
		jDayNo -= jDays[i]
		jM++
	}
	return jY, jM + 1, jDayNo + 1
}
