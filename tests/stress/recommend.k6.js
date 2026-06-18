// k6 stress test (proper load-testing tool artifact).
// Install k6: https://k6.io/docs/get-started/installation/
// Run against the live stack:
//   k6 run -e BASE=http://localhost:8080 tests/stress/recommend.k6.js
//
// Ramps to 50 virtual users and asserts p95 latency and error-rate thresholds.
import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.BASE || "http://localhost:8080";

export const options = {
  stages: [
    { duration: "20s", target: 50 }, // ramp up
    { duration: "40s", target: 50 }, // steady load
    { duration: "10s", target: 0 },  // ramp down
  ],
  thresholds: {
    http_req_failed: ["rate<0.05"],     // <5% errors
    http_req_duration: ["p(95)<500"],   // 95% under 500ms
  },
};

const users = ["user_123", "user_fashion", "user_foodie", "user_999"];

export default function () {
  const u = users[Math.floor(Math.random() * users.length)];
  const res = http.get(`${BASE}/api/v1/recommendations?user_id=${u}`);
  check(res, {
    "status is 200": (r) => r.status === 200,
    "has videos": (r) => {
      try { return JSON.parse(r.body).data.videos.length > 0; } catch { return false; }
    },
  });
  sleep(0.1);
}
