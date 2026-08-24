package account

import (
	"testing"

	"gaap-api/internal/logic/utils"

	"github.com/google/uuid"
)

func TestValidateAccountHierarchyAccess(t *testing.T) {
	parentID := uuid.New()
	tests := []struct {
		name     string
		plan     int
		isGroup  bool
		parentID uuid.UUID
		wantErr  bool
	}{
		{name: "free regular account", plan: utils.UserLevelFree},
		{name: "free account group", plan: utils.UserLevelFree, isGroup: true, wantErr: true},
		{name: "free child account", plan: utils.UserLevelFree, parentID: parentID, wantErr: true},
		{name: "unspecified account group", plan: utils.UserLevelUnspecified, isGroup: true, wantErr: true},
		{name: "unspecified child account", plan: utils.UserLevelUnspecified, parentID: parentID, wantErr: true},
		{name: "pro account group", plan: utils.UserLevelPro, isGroup: true},
		{name: "pro child account", plan: utils.UserLevelPro, parentID: parentID},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateAccountHierarchyAccess(test.plan, test.isGroup, test.parentID)
			if test.wantErr && err == nil {
				t.Fatal("expected access validation to fail")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected access validation to pass: %v", err)
			}
		})
	}
}
