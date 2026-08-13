package transaction

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTransactionNoteBoundaries(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateTransactionNote(strings.Repeat("界", maxTransactionNoteLength)))
	require.EqualError(
		t,
		validateTransactionNote(strings.Repeat("界", maxTransactionNoteLength+1)),
		"transaction note must not exceed 500 characters",
	)
}
