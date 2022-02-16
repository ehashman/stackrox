package signatures

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPublicKeyVerifier(t *testing.T) {
	cases := map[string]struct {
		base64EncKey string
		fail         bool
		err          error
	}{
		"valid public key": {
			base64EncKey: "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFMDRzb0Fv" +
				"TnlnUmhheXRDdHlnUGN3c1ArNkVpbgpZb0R2L0JKeDFUOVdtdHNBTmgySHBsUlI2NkZibSszT2pGdWFoMkloRnVmUGhEbDZhODVJM3l" +
				"tVll3PT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg==",
		},
		"error in decoding base64 encoded string": {
			base64EncKey: "<",
			fail:         true,
			err:          base64.CorruptInputError(0),
		},
		"non PEM encoded public key": {
			base64EncKey: "anVzdHNvbWV0ZXh0Cg==",
			fail:         true,
			err:          errox.InvariantViolation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := &storage.SignatureVerificationConfig_PublicKey{PublicKey: &storage.CosignPublicKeyVerification{PublicKeysBase64Enc: []string{c.base64EncKey}}}
			verifier, err := newPublicKeyVerifier(config)
			if c.fail {
				assert.Error(t, err)
				assert.Nil(t, verifier)
				assert.ErrorIs(t, err, c.err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, verifier.parsedPublicKeys, 1)
			}
		})
	}
}

func TestPublicKeyVerifier_VerifySignature_Success(t *testing.T) {
	const b64PublicKey = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFMDRzb0" +
		"FvTnlnUmhheXRDdHlnUGN3c1ArNkVpbgpZb0R2L0JKeDFUOVdtdHNBTmgySHBsUlI2NkZibSszT2pGdWFoMkloRnVmUGhEbDZhODVJM3ltV" +
		"ll3PT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="
	const b64Signature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAYZNkn8h" +
		"kAnXI="
	const b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4OTJkLTQ" +
		"4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODliMDA1Mj" +
		"hmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb250YWluZ" +
		"XIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	const imgString = "ttl.sh/d8d3892d-48bd-4671-a546-2e70a900b702@sha256:ee89b00528ff4f02f2405e4ee221743ebc3f8e8dd0" +
		"bfd5c4c20a2fa2aaa7ede3"

	pubKeyVerifier, err := newPublicKeyVerifier(&storage.SignatureVerificationConfig_PublicKey{
		PublicKey: &storage.CosignPublicKeyVerification{
			PublicKeysBase64Enc: []string{b64PublicKey}}})

	require.NoError(t, err, "creating public key verifier")

	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating image with signature")

	status, err := pubKeyVerifier.VerifySignature(context.Background(), img)
	assert.NoError(t, err, "verification should be successful")
	assert.Equal(t, storage.ImageSignatureVerificationResult_VERIFIED, status, "status should be VERIFIED")
}

func TestPublicKeyVerifier_VerifySignature_Failure(t *testing.T) {
	const b64NonMatchingPubKey = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0" +
		"FFV2kzdFN4dkJIN1MvV1VtdjQwOG5LUHhOU0p4NgordzdjOUZ0RlNrNmNveHgyVlViUHkvWDNVUzNjWGZrL3pWQStHN05iWEdCWWhBR2FPc" +
		"3BzNVpLamtRPT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="
	const b64Signature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAYZNkn8h" +
		"kAnXI="
	const b64SignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4OTJkLTQ" +
		"4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODliMDA1Mj" +
		"hmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb250YWluZ" +
		"XIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="
	const imgString = "ttl.sh/d8d3892d-48bd-4671-a546-2e70a900b702@sha256:ee89b00528ff4f02f2405e4ee221743ebc3f8e8dd0" +
		"bfd5c4c20a2fa2aaa7ede3"

	pubKeyVerifier, err := newPublicKeyVerifier(&storage.SignatureVerificationConfig_PublicKey{
		PublicKey: &storage.CosignPublicKeyVerification{
			PublicKeysBase64Enc: []string{b64NonMatchingPubKey}}})

	require.NoError(t, err, "creating public key verifier")

	emptyPubKeyVerifier, err := newPublicKeyVerifier(&storage.SignatureVerificationConfig_PublicKey{
		PublicKey: &storage.CosignPublicKeyVerification{}})
	require.NoError(t, err, "creating empty public key verifier")

	img, err := generateImageWithCosignSignature(imgString, b64Signature, b64SignaturePayload)
	require.NoError(t, err, "creating image with signature")

	cases := map[string]struct {
		verifier *publicKeyVerifier
		img      *storage.Image
		err      error
		status   storage.ImageSignatureVerificationResult_Status
	}{
		"fail with non-verifiable public key": {
			img:      img,
			verifier: pubKeyVerifier,
			status:   storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
		},
		"fail with empty public key verifier": {
			img:      img,
			verifier: emptyPubKeyVerifier,
			status:   storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
			err:      errNoKeysToVerifyAgainst,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			status, err := c.verifier.VerifySignature(context.Background(), c.img)
			assert.Equal(t, c.status, status, "status should be FAILED verification")
			assert.Error(t, err, "verification should be unsuccessful")
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			}
		})
	}
}

func TestRetrieveVerificationDataFromImage_Success(t *testing.T) {
	const b64CosignSignature = "MEUCIDGMmJyxVKGPxvPk/QlRzMSGzcI8pYCy+MB7RTTpegzTAiEArssqWntVN8oJOMV0Aey0zhsNqRmEVQAY" +
		"ZNkn8hkAnXI="
	const imgHash = "sha256:3a4d57227f02243dfc8a2849ec4a116646bed293b9e93cbf9d4a673a28ef6345"
	const b64CosignSignaturePayload = "eyJjcml0aWNhbCI6eyJpZGVudGl0eSI6eyJkb2NrZXItcmVmZXJlbmNlIjoidHRsLnNoL2Q4ZDM4O" +
		"TJkLTQ4YmQtNDY3MS1hNTQ2LTJlNzBhOTAwYjcwMiJ9LCJpbWFnZSI6eyJkb2NrZXItbWFuaWZlc3QtZGlnZXN0Ijoic2hhMjU2OmVlODli" +
		"MDA1MjhmZjRmMDJmMjQwNWU0ZWUyMjE3NDNlYmMzZjhlOGRkMGJmZDVjNGMyMGEyZmEyYWFhN2VkZTMifSwidHlwZSI6ImNvc2lnbiBjb25" +
		"0YWluZXIgaW1hZ2Ugc2lnbmF0dXJlIn0sIm9wdGlvbmFsIjpudWxsfQ=="

	img, err := generateImageWithCosignSignature("docker.io/nginx@"+imgHash, b64CosignSignature, b64CosignSignaturePayload)
	require.NoError(t, err, "error creating image")

	sigs, hash, err := retrieveVerificationDataFromImage(img)
	require.NoError(t, err, "should not fail")

	assert.Len(t, sigs, 1, "expected one signature")
	sig := sigs[0]
	b64sig, err := sig.Base64Signature()
	require.NoError(t, err)
	assert.Equal(t, b64CosignSignature, b64sig, "expected the base64 values of the signatures to match")
	payload, err := sig.Payload()
	require.NoError(t, err)
	expectedPayload, err := base64.StdEncoding.DecodeString(b64CosignSignaturePayload)
	require.NoError(t, err)
	assert.Equal(t, expectedPayload, payload, "expected the payloads of the signature to match")
	assert.Equal(t, imgHash, hash.String(), "expected the hash to match the image's hash")
}

func TestRetrieveVerificationDataFromImage_Failure(t *testing.T) {
	cases := map[string]struct {
		imgID string
		err   error
	}{
		"no image SHA": {
			err: errNoImageSHA,
		},
		"hash creation failed": {
			imgID: "invalid-hash",
			err:   errHashCreation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			img, err := generateImageWithCosignSignature("docker.io/nginx:latest", "", "")
			require.NoError(t, err, "error creating image")
			// Since we have no Image Metadata, the Image ID will be returned as SHA. This way we can test for invalid
			// SHA / no SHA.
			img.Id = c.imgID
			_, _, err = retrieveVerificationDataFromImage(img)
			assert.Error(t, err)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func generateImageWithCosignSignature(imgString, b64Sig, b64SigPayload string) (*storage.Image, error) {
	cimg, err := imgUtils.GenerateImageFromString(imgString)
	if err != nil {
		return nil, err
	}

	sigBytes, err := base64.StdEncoding.DecodeString(b64Sig)
	if err != nil {
		return nil, err
	}
	sigPayloadBytes, err := base64.StdEncoding.DecodeString(b64SigPayload)
	if err != nil {
		return nil, err
	}

	img := types.ToImage(cimg)
	img.Signature = &storage.ImageSignature{
		Signatures: []*storage.Signature{
			{
				Signature: &storage.Signature_Cosign{
					Cosign: &storage.CosignSignature{
						RawSignature:     sigBytes,
						SignaturePayload: sigPayloadBytes,
					},
				},
			},
		},
	}
	return img, nil
}
