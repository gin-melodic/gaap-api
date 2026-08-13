package account

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAccountMetadataBoundaries(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateAccountMetadata(strings.Repeat("界", maxAccountNameLength), strings.Repeat("N", maxAccountNumberLength), strings.Repeat("注", maxAccountRemarksLength)))
	require.EqualError(t, validateAccountMetadata(" ", "", ""), "account name is required")
	require.EqualError(t, validateAccountMetadata(strings.Repeat("界", maxAccountNameLength+1), "", ""), "account name must not exceed 100 characters")
	require.EqualError(t, validateAccountMetadata("valid", strings.Repeat("N", maxAccountNumberLength+1), ""), "account number must not exceed 50 characters")
	require.EqualError(t, validateAccountMetadata("valid", "", strings.Repeat("注", maxAccountRemarksLength+1)), "account remarks must not exceed 500 characters")
}
