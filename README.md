### ParseFlow

Concurrent Go log processing pipeline for Heroku HTTPS log drains
```
                    ┌──────────────────────┐
                    │   HTTP Receiver      │◄─── Heroku Logplex
                    │   POST /logdrains    │
                    └──────────┬───────────┘
                               │ raw logs
                               ▼
                    ┌──────────────────────┐
                    │    RawLogChan        │
                    └──────────┬───────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │   ParserWorker       │
                    └──────────┬───────────┘
                               │ ParsedLog structs
                               ▼
                    ┌──────────────────────┐
                    │   ParsedLogChan      │
                    └──────────┬───────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │     FanOut           │
                    └─────┬────────────┬───┘
                          │            │
           ┌──────────────▼─┐        ┌─▼─────────────────┐
           │   MetricChan   │        │  DbRawWriteChan   │
           └──────┬─────────┘        └─┬─────────────────┘
                  │                    │
                  ▼                    ▼
      ┌─────────────────────┐    ┌─────────────────────┐
      │ MetricsAggregator   |    |     DbWriter        |
      |  (Primary Actor)    |    │   (Sec Actor)       │
      └─────────┬───────────┘    └─┬───────────────────┘
                │                  │
                │                  ▼
                │            ┌─────────────────────┐
                │            │        DB           │
                │            └─────────────────────┘
                │
                ▼
      ┌─────────────────────┐
      │   Metrics API       │◄─── /Dashboard
      │   GET /metrics      │
      └─────────────────────┘
```
