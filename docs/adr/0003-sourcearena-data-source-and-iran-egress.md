# SourceArena as the sole market data source, requiring Iranian egress

All market data (options, symbols, underlying quote, candles, technical indicators) comes
from SourceArena. Two facts are surprising and not visible from the code alone, so they
are recorded here:

1. **Two hosts, two auth schemes.** The market API (`apis.sourcearena.ir`) takes the token
   as a query-string `?token=...`, while the candle API (`api3.sourcearena.ir`) takes it as
   an `X-Header-Token` header. Swapping them yields `invalid method`. See
   `SOURCEARENA_API.md`.
2. **Iranian network egress is mandatory.** Requests from outside Iran fail
   (`error_code: 1001` / dropped connections), so the service must run on Iranian internet
   or route through `SOURCEARENA_HTTP_PROXY` with Iranian egress. This is why the app runs
   fine but shows empty data in environments without Iranian egress.
