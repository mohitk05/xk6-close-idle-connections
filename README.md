# xk6-close-idle-connections
[k6](https://github.com/grafana/k6) extension to close TCP connections at a defined interval

## Build
```bash
xk6 build v0.53.0 --with github.com/mohitk05/xk6-close-idle-connections@latest
```

## Usage
```javascript
import * as closeIdleConnections from 'k6/x/closeIdleConnections';

export default function () {
  closeIdleConnections.start(10); // Will only init once, subsequent calls will return immediately
  // Your test script
}
```
