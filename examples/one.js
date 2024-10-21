import { sleep } from "k6";
import http from "k6/http";
import exec from "k6/execution";
import * as closeIdleConn from "k6/x/close_idle_conn";

export default function () {
    closeIdleConn.start(5);

    for (let i = 0; i < 12; i++) {
        const res = http.get("https://httpbin.test.k6.io/get");
        console.log(`Status ${res.status}, Scenario iteration ${exec.vu.iterationInInstance}, Iteration ${i}, http_req_connecting: ${res.timings.connecting}ms`);
        sleep(1);
    }

}

export function teardown() {
    closeIdleConn.end();
}