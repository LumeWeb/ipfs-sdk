package ipfs

import (
	"errors"
	"net/http"
	"testing"

	"github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Unwrap(t *testing.T) {
	base := notFoundErr("website not found").Error()
	apiErr := &APIError{Reason: ErrorCodeCIDNotPinned, Err: base}

	assert.True(t, errors.Is(apiErr, ErrNotFound), "APIError should unwrap to ErrNotFound")
	assert.ErrorIs(t, apiErr, base, "APIError should unwrap to the base error")
}

func TestErrorReasonOf(t *testing.T) {
	t.Run("extracts reason from APIError", func(t *testing.T) {
		base := plainErr("validation failed").Error()
		err := &APIError{Reason: ErrorCodeCIDNotPinned, Details: "target not pinned", Err: base}
		assert.Equal(t, ErrorCodeCIDNotPinned, ErrorReasonOf(err))
	})

	t.Run("extracts reason through unrecoverable wrap", func(t *testing.T) {
		base := plainErr("validation failed").Error()
		err := retry.Unrecoverable(&APIError{Reason: ErrorCodeIPNSKeyNotFound, Err: base})
		assert.Equal(t, ErrorCodeIPNSKeyNotFound, ErrorReasonOf(err))
	})

	t.Run("returns empty for plain errors", func(t *testing.T) {
		assert.Equal(t, "", ErrorReasonOf(errors.New("generic")))
		assert.Equal(t, "", ErrorReasonOf(nil))
	})
}

func TestHandleResponse_AttachesReason(t *testing.T) {
	body := []byte(`{"error":{"reason":"CID_NOT_PINNED","details":"target not pinned"}}`)

	err := handleResponse(http.StatusUnprocessableEntity, body, OpUpdateWebsite, []int{http.StatusOK})

	require.Error(t, err)
	assert.Equal(t, ErrorCodeCIDNotPinned, ErrorReasonOf(err))
	// Underlying sentinel should still match for the static mapping (422 -> plainErr).
	require.False(t, errors.Is(err, ErrNotFound))
}

func TestHandleResponse_GenericPathAttachesReason(t *testing.T) {
	// A status code with no static mapping should still surface the reason.
	body := []byte(`{"error":{"reason":"DNS_VALIDATION_FAILED"}}`)

	err := handleResponse(http.StatusBadRequest, body, OpVerifyWebsiteDomain, []int{http.StatusOK})

	require.Error(t, err)
	assert.Equal(t, ErrorCodeDNSValidationFailed, ErrorReasonOf(err))
}

func TestHandleResponse_NoReasonWhenBodyNotParsable(t *testing.T) {
	err := handleResponse(http.StatusBadRequest, []byte("plain text error"), OpGetWebsite, []int{http.StatusOK})

	require.Error(t, err)
	assert.Equal(t, "", ErrorReasonOf(err))
}

func TestHandleResponse_SuccessReturnsNil(t *testing.T) {
	err := handleResponse(http.StatusOK, []byte(`{"error":{"reason":"CID_NOT_PINNED"}}`), OpGetWebsite, []int{http.StatusOK})
	assert.NoError(t, err)
	assert.Equal(t, "", ErrorReasonOf(err))
}
