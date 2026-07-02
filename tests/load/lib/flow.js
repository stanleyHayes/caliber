import http from 'k6/http';
import { check } from 'k6';
import { urlFor } from '../config.js';
import { authHeader } from './auth.js';
import { runInterview } from './interview.js';

const ROLE_FREE_TEXT =
  'We need a senior Go backend engineer in Accra to own our matching services — ' +
  'must know Postgres and gRPC, ideally some Kubernetes. GHS 18k–25k, start within a month.';

export function runFlow(data) {
  const { empToken, empId, candToken, candId } = data;

  // Flow A: generate a role spec.
  const genResp = http.post(
    urlFor('/v1/roles:generate'),
    JSON.stringify({ employerId: empId, freeText: ROLE_FREE_TEXT }),
    {
      headers: authHeader(empToken),
      tags: { name: 'roles_generate' },
    }
  );
  check(genResp, {
    'roles:generate status is 200': (r) => r.status === 200,
    'roles:generate returns role.id': (r) => {
      const id = r.json('role.id');
      return id && id.length > 0;
    },
  });
  const roleId = genResp.json('role.id');

  // Flow A: shortlist for the generated role.
  const shortResp = http.get(
    urlFor(`/v1/roles/${roleId}/shortlist?page.page=1&page.page_size=5`),
    {
      headers: authHeader(empToken),
      tags: { name: 'roles_shortlist' },
    }
  );
  check(shortResp, {
    'shortlist status is 200': (r) => r.status === 200,
  });

  // Talent Radar reads.
  const radarEndpoints = ['pool', 'supply-demand', 'time-to-shortlist'];
  for (const endpoint of radarEndpoints) {
    const resp = http.get(
      urlFor(`/v1/radar/${endpoint}?page.page=1&page.page_size=5`),
      {
        headers: authHeader(empToken),
        tags: { name: `radar_${endpoint}` },
      }
    );
    check(resp, {
      [`radar/${endpoint} status is 200`]: (r) => r.status === 200,
    });
  }

  // Candidate profile read.
  const profileResp = http.get(
    urlFor(`/v1/candidates/${candId}/profile`),
    {
      headers: authHeader(candToken),
      tags: { name: 'candidate_profile' },
    }
  );
  check(profileResp, {
    'candidate profile status is 200': (r) => r.status === 200,
  });

  // Flow B: full interview via gRPC streaming.
  if (roleId && candId && candToken) {
    runInterview(candToken, candId, roleId);
  }

  // Flow C: agent time advance.
  const timeResp = http.post(
    urlFor(`/v1/candidates/${candId}/agent:timeAdvance`),
    '{}',
    {
      headers: authHeader(candToken),
      tags: { name: 'agent_timeAdvance' },
    }
  );
  check(timeResp, {
    'agent:timeAdvance status is 200': (r) => r.status === 200,
  });

  // Health check.
  const health = http.get(urlFor('/healthz'), {
    tags: { name: 'healthz' },
  });
  check(health, {
    'healthz status is 200': (r) => r.status === 200,
  });
}
