import { STAGES_LOAD, THRESHOLDS, EMP_EMAIL, CAND_EMAIL, PASSWORD } from '../config.js';
import { login } from '../lib/auth.js';
import { runFlow } from '../lib/flow.js';

export const options = {
  stages: STAGES_LOAD,
  thresholds: THRESHOLDS,
};

export function setup() {
  const emp = login(EMP_EMAIL, PASSWORD);
  const cand = login(CAND_EMAIL, PASSWORD);
  return {
    empToken: emp.token,
    empId: emp.userId,
    candToken: cand.token,
    candId: cand.userId,
  };
}

export default function (data) {
  runFlow(data);
}
