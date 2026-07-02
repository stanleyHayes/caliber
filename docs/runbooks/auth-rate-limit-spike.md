# Auth / rate-limit spike

**Symptoms:** Sudden increase in 401/403/429 responses, or `rate_limit_exceeded`
logs.

## Impact

- Legitimate users may be locked out (if auth provider is failing or keys are
  rotated).
- Aggressive clients may be consuming quota; 429s protect the API but can hide
  abuse.

## Triage

1. Check HTTP status distribution:

   ```promql
   sum by (status_code) (rate(http_server_duration_count[5m]))
   ```

2. Look for rate-limit events:

   ```logql
   {service="caliber-api"}
   | json
   | msg="rate_limit_exceeded"
   ```

3. Identify top clients / principals:

   ```logql
   {service="caliber-api"}
   | json
   | msg="rate_limit_exceeded"
   | line_format "{{.principal}} {{.client_ip}}"
   ```

4. Check JWT issuer/key endpoint health and token clock skew.

## Mitigation

- If a specific client is abusing the API, block or throttle the API key /
  principal temporarily.
- If legitimate users are failing auth, verify the JWKS endpoint, token
  expiration, and key rotation.
- If 429s are too aggressive for a planned traffic spike, raise the rate-limit
  bucket temporarily and document the change.
- Enable stricter WAF/IP rules if the traffic looks like an attack.

## Escalation

- Page security / platform if the pattern suggests a credential-stuffing or
  DDoS attack.
- Page the identity team if auth failures correlate with an identity provider
  incident.
