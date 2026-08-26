package demo_data

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestDemoSnapshotPreservesMoneyBoundariesAndRelationships(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	snapshots := []demoAccountSnapshot{
		{
			ID:             parentID,
			Name:           "Maximum balance",
			BalanceUnits:   math.MaxInt64,
			BalanceNanos:   999_999_999,
			DefaultChildID: childID,
		},
		{
			ID:           childID,
			ParentID:     parentID,
			Name:         "Negative boundary",
			BalanceUnits: math.MinInt64,
			BalanceNanos: -999_999_999,
		},
	}

	encoded, err := json.Marshal(snapshots)
	if err != nil {
		t.Fatalf("marshal snapshots: %v", err)
	}
	var decoded []demoAccountSnapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal snapshots: %v", err)
	}
	if decoded[0].BalanceUnits != math.MaxInt64 || decoded[0].BalanceNanos != 999_999_999 {
		t.Fatalf("maximum amount changed during snapshot: %+v", decoded[0])
	}
	if decoded[1].BalanceUnits != math.MinInt64 || decoded[1].BalanceNanos != -999_999_999 {
		t.Fatalf("negative amount changed during snapshot: %+v", decoded[1])
	}
	if decoded[0].DefaultChildID != childID || decoded[1].ParentID != parentID {
		t.Fatal("account relationships changed during snapshot")
	}
}
