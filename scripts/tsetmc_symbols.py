"""
tsetmc_symbols.py — یک‌بار کراول همه نمادهای tsetmc.com

خروجی:  data/symbols.json  و  data/symbols.csv
اجرا:   python3 scripts/tsetmc_symbols.py
نیاز:   requests (معمولاً از پیش نصب است روی Ubuntu)
"""

import csv
import json
import pathlib
import sys
import time

try:
    import requests
except ImportError:
    sys.exit("requests not installed — run: pip3 install requests")

HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/120.0.0.0 Safari/537.36"
    ),
    "Referer": "https://tsetmc.com/",
    "Accept": "application/json",
}

SEARCH_URL = "https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch/{term}"
MARKETWATCH_URL = (
    "https://cdn.tsetmc.com/api/ClosingPrice/GetMarketWatch"
    "?market=0&industry=0&isNoFilter=true&isAggregate=false"
)

PERSIAN_LETTERS = "ابپتثجچحخدذرزژسشصضطظعغفقکگلمنوهی"
FIELDS = ["ins_code", "symbol", "company_name", "isin", "instrument_id"]

SEARCH_LIMIT = 40  # API caps each search at 40 results


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
    """Search by term and add new instruments to seen dict. Returns how many added."""
    try:
        r = requests.get(SEARCH_URL.format(term=term), headers=HEADERS, timeout=20)
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
    return added


def fetch_market_watch(seen: dict) -> int:
    """Try market watch (works during market hours). Returns how many added."""
    try:
        r = requests.get(MARKETWATCH_URL, headers=HEADERS, timeout=30)
        r.raise_for_status()
        rows = r.json().get("marketwatch", []) or r.json().get("marketWatch", [])
    except Exception as e:
        print(f"  warn: market watch failed: {e}", flush=True)
        return 0
    added = 0
    for item in rows:
        inst = item.get("instrument", {})
        code = inst.get("insCode", "").strip()
        if not code or code in seen:
            continue
        seen[code] = {
            "ins_code": code,
            "symbol": inst.get("lVal18AFC", "").strip(),
            "company_name": inst.get("lVal30", "").strip(),
            "isin": inst.get("cIsin", "").strip(),
            "instrument_id": inst.get("instrumentID", "").strip(),
        }
        added += 1
    return added


def main() -> None:
    seen: dict[str, dict] = {}

    # Phase 1: market watch (works during trading hours)
    print("Phase 1: market watch …", flush=True)
    n = fetch_market_watch(seen)
    print(f"  → {n} from market watch (total: {len(seen)})", flush=True)

    # Phase 2: search by every single Persian letter
    print("Phase 2: single-letter searches …", flush=True)
    saturated: list[str] = []  # letters that hit the 40-result cap
    for ch in PERSIAN_LETTERS:
        before = len(seen)
        results_count = search(ch, seen)
        added = len(seen) - before
        if results_count >= SEARCH_LIMIT:
            saturated.append(ch)
        time.sleep(0.15)
    print(f"  → total after phase 2: {len(seen)}", flush=True)
    print(f"  → saturated letters (need 2-char drill): {''.join(saturated)}", flush=True)

    # Phase 3: two-character searches for saturated letters
    if saturated:
        print("Phase 3: two-char drill for saturated letters …", flush=True)
        for prefix in saturated:
            for ch in PERSIAN_LETTERS:
                term = prefix + ch
                search(term, seen)
                time.sleep(0.1)
            print(f"  → after '{prefix}*': {len(seen)}", flush=True)

    if not seen:
        sys.exit("No instruments found — check network/tsetmc availability")

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

    print(f"\nDone: {len(instruments)} instruments → {json_path}, {csv_path}")


if __name__ == "__main__":
    main()
