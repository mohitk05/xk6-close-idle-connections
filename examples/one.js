import { sleep } from "k6";
import http from "k6/http";
import { start } from "k6/x/close_idle_conn";

export default function () {
    start(5);

    for (let i = 0; i < 12; i++) {
        const res = http.get("http://test.k6.io");
        sleep(1);
        console.log(`Iteration ${i}, http_req_connecting: ${res.timings.connecting}ms`);
    }
}