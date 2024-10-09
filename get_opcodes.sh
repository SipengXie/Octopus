curl -X POST -H "Content-Type: application/json" --data '{
  "jsonrpc": "2.0",
  "method": "debug_traceTransaction",
  "params": [
    "0xaf37a7093d37b834a1f3cd04a03beb6c4dbb545bdb43fcaa8a3be161e5c0de5a",
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