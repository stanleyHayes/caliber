import { apiFetch } from './client';

// The authenticated candidate's complete data export (DSAR, CAL-118). The subject
// is taken from the access token server-side; document is a JSON string.
export interface DataExportResponse {
  document: string;
}

export const privacyApi = {
  exportMyData: () => apiFetch<DataExportResponse>('/v1/me/data'),
};
