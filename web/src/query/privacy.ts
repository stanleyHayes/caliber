import { useMutation } from '@tanstack/react-query';

import { privacyApi } from '../api/privacy';

// useExportMyData fetches the candidate's full data export on demand (a DSAR is a
// deliberate user action, so it is a mutation rather than a background query).
export function useExportMyData() {
  return useMutation({ mutationFn: () => privacyApi.exportMyData() });
}
