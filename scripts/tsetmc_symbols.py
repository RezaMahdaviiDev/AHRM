"""
tsetmc_symbols.py — کراول همه نمادهای tsetmc.com با ISIN

خروجی:  data/symbols.json  و  data/symbols.csv
اجرا:   python3 scripts/tsetmc_symbols.py
نیاز:   requests (معمولاً از پیش نصب است روی Ubuntu)

مراحل:
  1. market watch (اگر بازار باز باشد)
  2. جستجوی تک‌حرفی با همه حروف فارسی
  3. جستجوی دو‌حرفی برای حروف با نتایج ≥40
  4. enrichment: دریافت ISIN از GetInstrumentInfo برای هر insCode
"""

import csv
import json
import pathlib
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

try:
    import requests
except ImportError:
    sys.exit("requests not installed — run: pip3 install requests")

SESSION = requests.Session()
SESSION.headers.update({
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/120.0.0.0 Safari/537.36"
    ),
    "Referer": "https://tsetmc.com/",
    "Accept": "application/json",
})

SEARCH_URL    = "https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch/{term}"
INFO_URL      = "https://cdn.tsetmc.com/api/Instrument/GetInstrumentInfo/{ins_code}"
MARKETWATCH_URL = (
    "https://cdn.tsetmc.com/api/ClosingPrice/GetMarketWatch"
    "?market=0&industry=0&isNoFilter=true&isAggregate=false"
)

PERSIAN_LETTERS = "ابپتثجچحخدذرزژسشصضطظعغفقکگلمنوهی"
FIELDS = ["ins_code", "symbol", "company_name", "isin", "instrument_id"]
SEARCH_LIMIT = 40
ENRICH_WORKERS = 8
ENRICH_DELAY  = 0.12  # seconds between requests per worker


def _s(v) -> str:
    return (v or "").strip()


def parse_search_item(item: dict) -> dict | None:
    code = _s(item.get("insCode"))
    if not code:
        return None
    return {
        "ins_code": code,
        "symbol": _s(item.get("lVal18AFC")),
        "company_name": _s(item.get("lVal30")),
        "isin": _s(item.get("cIsin")),
        "instrument_id": _s(item.get("instrumentID")),
    }


def search(term: str, seen: dict) -> int:
    try:
        r = SESSION.get(SEARCH_URL.format(term=term), timeout=20)
        r.raise_for_status()
        items = r.json().get("instrumentSearch", [])
    except Exception as e:
        print(f"  warn: search '{term}' failed: {e}", flush=True)
        return 0
    added = 0
    for item in items:
        parsed = parse_search_item(item)
        if parsed and parsed["ins_code"] not in seen:
            seen[parsed["ins_code"]] = parsed
            added += 1
    return len(items)  # return raw count to detect saturation


def fetch_market_watch(seen: dict) -> int:
    try:
        r = SESSION.get(MARKETWATCH_URL, timeout=30)
        r.raise_for_status()
        rows = r.json().get("marketwatch", []) or r.json().get("marketWatch", [])
    except Exception as e:
        print(f"  warn: market watch failed: {e}", flush=True)
        return 0
    added = 0
    for item in rows:
        inst = item.get("instrument", {})
        code = _s(inst.get("insCode"))
        if not code or code in seen:
            continue
        seen[code] = {
            "ins_code": code,
            "symbol": _s(inst.get("lVal18AFC")),
            "company_name": _s(inst.get("lVal30")),
            "isin": _s(inst.get("cIsin")),
            "instrument_id": _s(inst.get("instrumentID")),
        }
        added += 1
    return added


def fetch_isin(ins_code: str) -> tuple[str, str, str]:
    """Fetch ISIN and instrumentID for one insCode. Returns (ins_code, isin, instrument_id)."""
    try:
        time.sleep(ENRICH_DELAY)
        r = SESSION.get(INFO_URL.format(ins_code=ins_code), timeout=20)
        r.raise_for_status()
        info = r.json().get("instrumentInfo", {})
        return ins_code, _s(info.get("cIsin")), _s(info.get("instrumentID"))
    except Exception:
        return ins_code, "", ""


def enrich_isin(seen: dict) -> None:
    """Phase 4: fill in ISIN + instrumentID from GetInstrumentInfo for each instrument."""
    codes = [code for code, v in seen.items() if not v["isin"]]
    if not codes:
        print("  → all instruments already have ISIN", flush=True)
        return
    print(f"  fetching ISIN for {len(codes)} instruments ({ENRICH_WORKERS} workers) …", flush=True)
    done = 0
    with ThreadPoolExecutor(max_workers=ENRICH_WORKERS) as pool:
        futures = {pool.submit(fetch_isin, code): code for code in codes}
        for fut in as_completed(futures):
            code, isin, instrument_id = fut.result()
            seen[code]["isin"] = isin
            seen[code]["instrument_id"] = instrument_id
            done += 1
            if done % 100 == 0 or done == len(codes):
                pct = done * 100 // len(codes)
                with_isin = sum(1 for v in seen.values() if v["isin"])
                print(f"  [{pct:3d}%] {done}/{len(codes)} fetched — {with_isin} have ISIN", flush=True)


def main() -> None:
    seen: dict[str, dict] = {}

    print("Phase 1: market watch …", flush=True)
    n = fetch_market_watch(seen)
    print(f"  → {n} from market watch (total: {len(seen)})", flush=True)

    print("Phase 2: single-letter searches …", flush=True)
    saturated: list[str] = []
    for ch in PERSIAN_LETTERS:
        count = search(ch, seen)
        if count >= SEARCH_LIMIT:
            saturated.append(ch)
        time.sleep(0.15)
    print(f"  → total: {len(seen)} — saturated: {''.join(saturated)}", flush=True)

    if saturated:
        print("Phase 3: two-char drill for saturated letters …", flush=True)
        for prefix in saturated:
            for ch in PERSIAN_LETTERS:
                search(prefix + ch, seen)
                time.sleep(0.1)
            print(f"  → after '{prefix}*': {len(seen)}", flush=True)

    if not seen:
        sys.exit("No instruments found — check network/tsetmc availability")

    print(f"Phase 4: enriching ISIN …", flush=True)
    enrich_isin(seen)

    instruments = sorted(seen.values(), key=lambda x: x["symbol"])

    out_dir = pathlib.Path("data")
    out_dir.mkdir(exist_ok=True)

    json_path = out_dir / "symbols.json"
    json_path.write_text(
        json.dumps(instruments, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )

    csv_path = out_dir / "symbols.csv"
    with csv_path.open("w", newline="", encoding="utf-8-sig") as f:
        w = csv.DictWriter(f, fieldnames=FIELDS)
        w.writeheader()
        w.writerows(instruments)

    with_isin = sum(1 for x in instruments if x["isin"])
    print(f"\nDone: {len(instruments)} instruments, {with_isin} with ISIN → {json_path}, {csv_path}")


if __name__ == "__main__":
    main()
