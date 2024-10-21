# xk6-close-idle-connections
[k6](https://github.com/grafana/k6) extension to close TCP connections at a defined interval

## Build
```bash
xk6 build v0.53.0 --with github.com/mohitk05/xk6-close-idle-connections@latest
```

## Usage
```javascript
import * as closeIdleConnections from 'k6/x/close_idle_conn';

export default function () {
  closeIdleConnections.start(10); // Will only init once for a VU, subsequent calls will return immediately
  // Your test script
}

export function teardown() {
  closeIdleConnections.stop();
}
```
