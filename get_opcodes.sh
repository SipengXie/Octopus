curl -X POST -H "Content-Type: application/json" --data '{
  "jsonrpc": "2.0",
  "method": "debug_traceTransaction",
  "params": [
    "0x82491091221b3be74ddb85b9351e2dabc8c77582cad4783d8f85dd0ff0d2b73f",
    {
      "tracerConfig": {
        "enableMemory": true,
        "enableReturnData": false,
        "enableStack": true,
        "enableStorage": true
      }
    }
  ],
  "id": 0
}' localhost:8545 > op.json