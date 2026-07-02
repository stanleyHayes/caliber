/**
 * Shared configuration for Caliber k6 load tests (CAL-142).
 */

export const REST_ADDR = __ENV.REST_ADDR || 'http://localhost:8080';
export const GRPC_ADDR = __ENV.GRPC_ADDR || 'localhost:9090';
export const EMP_EMAIL = __ENV.EMP_EMAIL || 'talent@mtn.com.gh';
export const CAND_EMAIL = __ENV.CAND_EMAIL || 'ama.mensah@example.com';
export const PASSWORD = __ENV.PASSWORD || 'Demo-Caliber-2026';

export const THRESHOLDS = {
  // Matches the HTTP error-rate SLO (<1%) and availability target.
  http_req_failed: ['rate<0.01'],
  // Matches the HTTP p95 latency SLO (<2s).
  http_req_duration: ['p(95)<2000'],
};

export const STAGES_SMOKE = [
  { duration: '5s', target: 2 },
  { duration: '60s', target: 2 },
  { duration: '5s', target: 0 },
];

export const STAGES_LOAD = [
  { duration: '1m', target: 20 },
  { duration: '2m', target: 20 },
  { duration: '30s', target: 0 },
];

export const DEFAULT_OPTIONS = {
  thresholds: THRESHOLDS,
};

export function urlFor(path) {
  return `${REST_ADDR}${path}`;
}
