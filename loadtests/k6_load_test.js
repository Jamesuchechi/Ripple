import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 },  // Ramp-up to 50 VUs
    { duration: '30s', target: 200 }, // Sustain 200 VUs load
    { duration: '10s', target: 0 },   // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<100'], // p95 < 50ms, p99 < 100ms
    http_req_failed: ['rate<0.01'],              // Error rate < 1%
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'rip_live_testkey';

export default function () {
  const headers = {
    Authorization: `Bearer ${API_KEY}`,
    'Content-Type': 'application/json',
  };

  // Scenario 1: Ingest Activity Event
  const payload = JSON.stringify({
    verb: 'post.created',
    actor_id: `user_${Math.floor(Math.random() * 1000)}`,
    object_id: `post_${Math.floor(Math.random() * 50000)}`,
  });

  const ingestRes = http.post(`${BASE_URL}/v1/events`, payload, { headers });
  check(ingestRes, {
    'ingest status is 200/201': (r) => r.status === 200 || r.status === 201,
  });

  // Scenario 2: Query Recipient Feed
  const recipientID = `user_${Math.floor(Math.random() * 1000)}`;
  const feedRes = http.get(`${BASE_URL}/v1/feed/${recipientID}?limit=20`, { headers });
  check(feedRes, {
    'feed query status is 200': (r) => r.status === 200,
  });

  sleep(0.1);
}
