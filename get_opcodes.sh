curl -X POST -H "Content-Type: application/json" --data '{
  "jsonrpc": "2.0",
  "method": "debug_traceTransaction",
  "params": [
    "0xfc729b2406f5ac3a8cafb58e9d65cd7cab30cb8fddbd31ac0b9bc162bfa6f6c0",
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