# ADR 0007: Alert Dedup — یک الارم در روز برای هر فرصت

**تاریخ:** ۲۰۲۶-۰۶-۲۸  
**وضعیت:** پذیرفته‌شده  
**نسخه:** v0.5.1

---

## مشکل

پس از فعال‌سازی Bale alerts در v0.5.0، مشاهده شد که در یک روز معاملاتی:
- **۲۳۱ پیام** برای `bale_bull_spread` ارسال شد (همان جفت‌ها هر ۳ دقیقه)
- **۳۲ پیام** برای `matrix` (همان قانون هر ۳ دقیقه)

کاربر انتظار داشت هر فرصت **یک بار در روز** اطلاع‌رسانی شود.

---

## ریشه مشکل

**علت اول — کلید dedup شامل مقدار لحظه‌ای بود:**

```go
// بول اسپرد: key شامل R با دقت ۲ رقم اعشار
key := fmt.Sprintf("bale-bs:%s:%s:%s:%.2f", kind, expiry, K2Symbol, R)
// matrix: key شامل diff
key := fmt.Sprintf("matrix:%s:%.0f", ruleID, diff)
```

قیمت آپشن‌ها هر ۳ دقیقه (با هر snapshot) تغییر می‌کند. R از ۲.۵۷ به ۲.۵۲ می‌رود
→ کلید جدید → dedup bypass → پیام جدید. همان جفت ضهرم5033/ضهرم5036 امروز
بارها با R‌های مختلف (۲.۸۷، ۲.۹۴، ۲.۸۸، ...) پیام داد.

**علت دوم — WasSent بدون محدودیت زمانی:**

```sql
SELECT COUNT(*) FROM alert_history WHERE alert_type = ? AND alert_key = ?
```

Pruning (حذف رکوردهای قدیمی) فقط در startup اجرا می‌شد. اگر سرویس چند روز بدون
restart بود، رکوردهای روزهای قبل می‌ماندند و الارم‌های مشروع روز جدید را block
می‌کردند.

---

## تصمیم

**هویت هر فرصت** با مشخصات ثابت آن تعریف می‌شود، نه با مقدار لحظه‌ای:

| نوع الارم | کلید قدیم | کلید جدید |
|---|---|---|
| بول اسپرد | `bale-bs:{kind}:{expiry}:{K2}:{R:.2f}` | `bale-bs:{kind}:{expiry}:{K2}` |
| ماتریس | `matrix:{ruleID}:{diff:.0f}` | `matrix:{ruleID}` |
| آربیتراژ R12 | `bale-arb-r12:{expiry}:{strike:.0f}:{pct:.2f}` | `bale-arb-r12:{expiry}:{strike:.0f}` |
| کاورد کال | `bale-cc-roi:{expiry}:{strike:.0f}:{roi:.2f}` | `bale-cc-roi:{expiry}:{strike:.0f}` |
| Breadth | `breadth:{state}:{avg:.4f}` | `breadth:{state}` |
| Advance/Decline | `ad:{state}:{avg:.4f}` | `ad:{state}` |

`WasSent` محدود به ۲۴ ساعت اخیر:

```sql
SELECT COUNT(*) FROM alert_history 
WHERE alert_type = ? AND alert_key = ? 
AND sent_at > datetime('now', '-24 hours')
```

---

## پیامدها

- هر جفت/آپشن/قانون: **یک پیام در روز**، صرف‌نظر از نوسان قیمت
- فردا: همان فرصت‌ها مجدداً ترگر می‌شوند (۲۴ ساعت گذشته)
- اگر R یک بول اسپرد از ۲.۵ به ۴.۰ برسد، پیام جدیدی نمی‌آید — کاربر باید صفحه
  را نگاه کند تا مقدار به‌روز را ببیند (trade-off آگاهانه)
- Startup pruning باقی می‌ماند به‌عنوان cleanup اختیاری (بی‌ضرر)
