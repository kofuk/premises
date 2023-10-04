# Monitoring

You can monitor status on port 8521.
TLS 1.3 is required to connect to the port.

## Endpoints

All endpoints requires`X-Auth-Key` header is set to auth key written in config file.

### `/monitor`

Monitor servers's status via Web Socket.

```json
{
    "status": "Loading...",
    "shutdown": false
}
```

`true` of `shutdown` means that server stopped.
After this payload sent, connection will be discarded.

### `/stop`

Stops server.
