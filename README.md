## ParseFlow

A tool to ingest heroku logdrains,parse, visualize and store. With failure rate tracking and slow requests flagging
```
                    ┌──────────────────────┐
                    │    Logs Receiver     │
                    │  (HTTPS Endpoint)    │
                    └──────────┬───────────┘
                               │
                            raw logs
                               ▼
                      ┌──────────────────┐
                      │   rawLogChan     │ (chan)
                      └────────┬─────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
┌───────────────┐     ┌───────────────┐      ┌───────────────┐
│ ParserWorker1 │     │ ParserWorker2 │  ..  │ ParseWorkerN  │
└──────┬────────┘     └──────┬────────┘      └──────┬────────┘
       │                     │                      │
       └─────────────┬───────┴──────────────────────┘
                     ▼
                ┌────────────────────────┐
                │    parsedLogChan       │ (chan LogEntry)
                └────────┬───────────────┘
                         │
        ┌────────────────┼────────────────┐
        ▼                                 ▼
┌──────────────────┐             ┌──────────────────┐
│   BatchWriter    │             │     Visualizer   │
└────────┬─────────┘             └────────┬─────────┘
         │                                │
         │ batch writes                   │ 
         ▼                                ▼
┌────────────────────┐            ┌─────────────────┐
│     Database       │            │ Live Dashboard  │
└────────────────────┘            └─────────────────┘
```