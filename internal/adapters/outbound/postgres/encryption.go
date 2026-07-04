package postgres

import (
	"encoding/json"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
)

// encodeEncryptedJSON marshals v for storage in a JSONB column, encrypting the
// payload when a field cipher is configured (CAL-117). In passthrough mode the
// object is stored as-is, byte-identical to legacy rows; with a key the ciphertext
// is wrapped as a JSON string so the column stays valid JSON while the PII inside
// (CV evidence quotes, intake, report-card evidence) is opaque at rest.
func encodeEncryptedJSON(cipher *fieldcrypto.FieldCipher, v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if !cipher.Enabled() {
		return raw, nil
	}
	enc, err := cipher.Encrypt(string(raw))
	if err != nil {
		return nil, err
	}
	return json.Marshal(enc)
}

// decodeEncryptedJSON reverses encodeEncryptedJSON into out. A non-empty JSONB
// string value is a (possibly encrypted) wrapped payload; a JSONB object is the
// legacy/passthrough form stored directly. A JSONB null (or empty string) also
// decodes into a Go string but carries no payload, so it leaves out untouched
// rather than being mis-read as an undecryptable wrapped value. The domain
// structs these decode into never marshal to a bare JSON string, so object vs.
// wrapped-string is unambiguous.
func decodeEncryptedJSON(cipher *fieldcrypto.FieldCipher, stored []byte, out any) error {
	if len(stored) == 0 {
		return nil
	}
	var wrapped string
	if err := json.Unmarshal(stored, &wrapped); err == nil {
		if wrapped == "" {
			return nil
		}
		plain, derr := cipher.Decrypt(wrapped)
		if derr != nil {
			return derr
		}
		return json.Unmarshal([]byte(plain), out)
	}
	return json.Unmarshal(stored, out)
}
