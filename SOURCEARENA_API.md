# SourceArena API — راهنمای داخلی پروژه AHRM

این فایل خلاصه‌ی تجربی و تست‌شده‌ی نحوه‌ی احراز هویت و فراخوانی API های SourceArena است که در این پروژه استفاده می‌شوند. هدف این است که هر بار نیازی به دیباگ دوباره نباشد.

## نکته‌ی کلیدی: دو هاست متفاوت با دو روش احراز هویت متفاوت

SourceArena دو دسته API دارد که روی دو دامنه‌ی متفاوت هستند و **روش پاس دادن توکن در هرکدام فرق دارد**. اشتباه گرفتن این دو، باعث خطای `{"Error":"invalid method"}` می‌شود.

### ۱. API بازار (Options / Symbols / Underlying) — `apis.sourcearena.ir`

- Base URL: `https://apis.sourcearena.ir/api/`
- **احراز هویت: توکن باید در query string باشد** (`?token=XXX`)
- هدر `X-Header-Token` روی این هاست کار **نمی‌کند** و خطای `invalid method` می‌دهد.

نمونه‌های تست‌شده و سالم:

```
GET https://apis.sourcearena.ir/api/?token=XXX&all=e
GET https://apis.sourcearena.ir/api/?token=XXX&all&type=2
GET https://apis.sourcearena.ir/api/?token=XXX&name=اهرم
```

| Endpoint | پارامترها | توضیح |
|---|---|---|
| Options | `?token=XXX&all=e` | لیست همه آپشن‌ها |
| All Symbols | `?token=XXX&all&type=2` | همه نمادها (بورس، فرابورس، صندوق، آپشن و ...) |
| Single Symbol / Underlying | `?token=XXX&name=<symbol>` | اطلاعات یک نماد خاص |

### ۲. API کندل (Candle / HV) — `api3.sourcearena.ir`

- Base URL: `https://api3.sourcearena.ir/api/v2/candle/1m`
- **احراز هویت: هدر `X-Header-Token`** (نه query string)

نمونه‌ی تست‌شده:

```bash
curl.exe -s -H "X-Header-Token:XXX" \
  "https://api3.sourcearena.ir/api/v2/candle/1m?from=1700000000&to=1710000000&symbol=اهرم&resolution=1D&type=1"
```

| پارامتر | مقدار |
|---|---|
| `from` / `to` | unix timestamp |
| `resolution` | `1, 15, 30, 60, 120, 240, 1D, 1W, 1M` |
| `type` | نوع تعدیل (جدول پایین) |

### انواع تعدیل (`type`) برای candle API

| مقدار | معنی |
|---|---|
| 0 | تعدیل‌نشده |
| 1 | افزایش سرمایه و سود نقدی |
| 2 | افزایش سرمایه |
| 3 | سود نقدی |
| 4 | عملکردی |

## پیاده‌سازی در پروژه

فایل: `internal/sourcearena/client.go`

- `getMarket()` → برای options/symbols/underlying، توکن را در query string اضافه می‌کند (`marketURL()`).
- `getCandle()` → برای candle، توکن را به‌صورت هدر `X-Header-Token` می‌فرستد.
- این دو تابع و auth آن‌ها **کاملاً مستقل** از هم هستند و نباید با هم یکی شوند.

## فرمول HV

```
HV = σ(ln returns) × 15.8 × 100
```

- بازه‌ی درخواست کندل: ۱۸۰ روز تقویمی (`hvCandleLookbackDays` در `internal/scanner/service.go`)
- حداقل تعداد کندل لازم برای محاسبه: ۴۱ کندل (به همین دلیل بازه‌ی ۹۰ روزه کافی نبود و به ۱۸۰ روز افزایش یافت — تعطیلات/آخر هفته بازار را در نظر بگیرید)

## نکات تست محلی (ویندوز)

اگر API از خارج از ایران فراخوانی شود، خطای `error_code: 1001` یا `توکن وب سرویس را وارد کنید` می‌دهد یا کانکشن قطع می‌شود. تست واقعی باید روی اینترنت ایران (یا با پروکسی ایرانی) انجام شود.

تست سریع با curl:

```powershell
$token = (Get-Content .env -Encoding UTF8 | Where-Object { $_ -match '^SOURCEARENA_API_TOKEN=' }) -replace '^SOURCEARENA_API_TOKEN=',''

# Market API
curl.exe -s "https://apis.sourcearena.ir/api/?token=$token&all=e"
curl.exe -s "https://apis.sourcearena.ir/api/?token=$token&all&type=2"
curl.exe -s "https://apis.sourcearena.ir/api/?token=$token&name=اهرم"

# Candle API
curl.exe -s -H "X-Header-Token:$token" "https://api3.sourcearena.ir/api/v2/candle/1m?from=1700000000&to=1710000000&symbol=اهرم&resolution=1D&type=1"
```

پاسخ سالم: آرایه/آبجکت JSON با داده‌های واقعی.
پاسخ ناسالم: `{"Error":"invalid method"}` یا `{"success":false,"message":"...","error_code":...}`.

---

**آخرین بروزرسانی:** بر اساس تست موفق در ۲۰۲۶/۰۶/۱۱ — هر دو API (market و candle) با موفقیت کار می‌کنند و صفحات `/hv` و `/arbitrage` بدون خطا لود می‌شوند.






1. وضعیت API ها



| #   | عنوان                     | آدرس                                                                                                        | آخرین بررسی     | وضعیت |
| --- | ------------------------- | ----------------------------------------------------------------------------------------------------------- | --------------- | ----- |
| 30  | ارز و سکه (v2)            | /api/?currency&v2                                                                                           | 1404-12-9 10:45 | OK    |
| 29  | پلن کندل                  | https://api3.sourcearena.ir/api/v2/candle/1m?from=1717273799&to=1747273799&symbol=فملی&resolution=1D&type=1 | 1404-12-9 10:45 | OK    |
| 26  | نماد های بسته             | /api/?closed_symbols                                                                                        | 1404-12-9 10:45 | OK    |
| 23  | تعدیل شده                 | /api/?adjusted&name=فملی&from=13800108&to=14011028&type=1                                                   | 1404-12-9 10:45 | OK    |
| 22  | اختیار معامله             | /api/?all=e                                                                                                 | 1404-12-9 10:45 | OK    |
| 21  | اختیار معامله (ویژه)      | /api/?custom_tradeoption                                                                                    | 1404-12-9 10:45 | OK    |
| 20  | کندل روزانه               | /api/?name=فملی&days=15                                                                                     | 1404-12-9 10:45 | OK    |
| 19  | پیام ناظر                 | /api/?inspect=all                                                                                           | 1404-12-9 10:45 | OK    |
| 18  | سهامداران                 | /api/?stockholder=شستا                                                                                      | 1404-12-9 10:45 | OK    |
| 17  | شاخص بورس                 | /api/?market=market_bourse                                                                                  | 1404-12-9 10:45 | OK    |
| 15  | اطلاعیه های کدال          | /api/?codal=شتران&p=1                                                                                       | 1404-12-9 10:45 | OK    |
| 14  | کندل 2 دقیقه ای           | /api/?intra_day=فملی                                                                                        | 1405-3-17 12:15 | OK    |
| 13  | ریز معاملات               | /api/?trades=فملی                                                                                           | 1405-3-17 12:15 | OK    |
| 11  | NAV صندوق                 | /api/?nav=all                                                                                               | 1405-3-17 12:15 | OK    |
| 10  | شاخص صنایع                | /api/?market=indices                                                                                        | 1405-3-17 12:15 | OK    |
| 8   | ارز و سکه                 | /api/?currency                                                                                              | 1405-3-17 12:15 | OK    |
| 7   | قیمت خودرو                | /api/?car=all                                                                                               | 1405-3-17 12:15 | OK    |
| 6   | اندیکاتور ها (تعدیل شده)  | /api/?all_indicators&name=فملی&adjusted=1                                                                   | 1405-3-17 12:15 | OK    |
| 5   | اندیکاتور ها (بدون تعدیل) | /api/?all_indicators&name=فملی                                                                              | 1405-3-17 12:15 | OK    |
| 4   | نماد خاص                  | /api/?name=فملی                                                                                             | 1405-3-17 12:15 | OK    |
| 3   | ارزهای دیجیتال            | /api/?crypto_v2=all                                                                                         | 1405-3-17 12:15 | OK    |
| 2   | همه نمادهای بورس          | /api/?all                                                                                                   | 1405-3-17 12:15 | OK    |

