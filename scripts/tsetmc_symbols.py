"""
tsetmc_symbols.py — یک‌بار کراول همه نمادهای tsetmc.com

خروجی:  data/symbols.json  و  data/symbols.csv
اجرا:   python scripts/tsetmc_symbols.py
نیاز:   pip install requests
"""

import csv
import json
import pathlib
import sys
import time

try:
    import requests
except ImportError:
    sys.exit("requests not installed — run: pip install requests")

HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/120.0.0.0 Safari/537.36"
    ),
    "Referer": "https://tsetmc.com/",
    "Accept": "application/json",
}

CDN_URL = (
    "https://cdn.tsetmc.com/api/ClosingPrice/GetMarketWatch"
    "?market=0&industry=0&isNoFilter=true&isAggregate=false"
)

FIELDS = ["ins_code", "symbol", "company_name", "isin", "instrument_id"]


def fetch_cdn() -> dict:
    print("Fetching from CDN JSON API …")
    r = requests.get(CDN_URL, headers=HEADERS, timeout=30)
    r.raise_for_status()
    data = r.json()
    rows = data.get("marketWatch", [])
    seen: dict[str, dict] = {}
    for item in rows:
        inst = item.get("instrument", {})
        code = inst.get("insCode", "").strip()
        if not code:
            continue
        seen[code] = {
            "ins_code": code,
            "symbol": inst.get("lVal18AFC", "").strip(),
            "company_name": inst.get("lVal30", "").strip(),
            "isin": inst.get("cIsin", "").strip(),
            "instrument_id": inst.get("instrumentID", "").strip(),
        }
    return seen


def fetch_old_api() -> dict:
    """Fallback: old pipe-separated market watch endpoint."""
    print("CDN failed — trying old text API …")
    url = "http://www.tsetmc.com/tsev2/data/MarketWatchPlus.aspx?h=0&r=0"
    r = requests.get(url, headers=HEADERS, timeout=30)
    r.raise_for_status()
    text = r.text
    seen: dict[str, dict] = {}
    # Format: sections separated by "@"; first section is instrument list
    # Each row: insCode,isin,instrumentID,lVal18AFC,lVal30,...
    section = text.split("@")[0] if "@" in text else text
    for line in section.splitlines():
        parts = line.split(",")
        if len(parts) < 5:
            continue
        code = parts[0].strip()
        if not code:
            continue
        seen[code] = {
            "ins_code": code,
            "isin": parts[1].strip() if len(parts) > 1 else "",
            "instrument_id": parts[2].strip() if len(parts) > 2 else "",
            "symbol": parts[3].strip() if len(parts) > 3 else "",
            "company_name": parts[4].strip() if len(parts) > 4 else "",
        }
    return seen


def main() -> None:
    seen: dict[str, dict] = {}

    try:
        seen = fetch_cdn()
    except Exception as e:
        print(f"CDN error: {e}")
        try:
            seen = fetch_old_api()
        except Exception as e2:
            sys.exit(f"Both endpoints failed: {e2}")

    if not seen:
        sys.exit("No instruments returned — check network or tsetmc availability")

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

    print(f"Fetched {len(instruments)} instruments → {json_path}, {csv_path}")


if __name__ == "__main__":
    main()
