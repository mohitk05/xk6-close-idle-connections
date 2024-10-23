# xk6-close-idle-connections
[k6](https://github.com/grafana/k6) extension to close idle TCP connections (per VU) at a defined interval.

## Build
```bash
xk6 build v0.53.0 --with github.com/mohitk05/xk6-close-idle-connections@latest
```

## Usage
```javascript
import * as closeIdleConnections from 'k6/x/close_idle_conn';

export default function () {
  // Will only run once for a VU, subsequent calls will return immediately
  closeIdleConnections.start(10); // closes idle connections every 10 seconds
  // Your test script
}

export function teardown() {
  closeIdleConnections.stop();
}
```
