package v1

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON implements json.Marshaler for TaskType
// It marshals the enum as an integer instead of a string
func (t TaskType) MarshalJSON() ([]byte, error) {
	return json.Marshal(int32(t))
}

// UnmarshalJSON implements json.Unmarshaler for TaskType
// It accepts both integer and string representations
func (t *TaskType) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as integer first
	var intVal int32
	if err := json.Unmarshal(data, &intVal); err == nil {
		*t = TaskType(intVal)
		return nil
	}

	// Try to unmarshal as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return fmt.Errorf("TaskType must be an integer or string, got: %s", string(data))
	}

	// Parse string value
	switch strVal {
	case "TASK_TYPE_UNSPECIFIED":
		*t = TaskType_TASK_TYPE_UNSPECIFIED
	case "TASK_TYPE_ACCOUNT_MIGRATION":
		*t = TaskType_TASK_TYPE_ACCOUNT_MIGRATION
	case "TASK_TYPE_EXPORT_DATA":
		*t = TaskType_TASK_TYPE_EXPORT_DATA
	case "TASK_TYPE_IMPORT_DATA":
		*t = TaskType_TASK_TYPE_IMPORT_DATA
	default:
		return fmt.Errorf("unknown TaskType: %s", strVal)
	}
	return nil
}

// MarshalJSON implements json.Marshaler for TaskStatus
// It marshals the enum as an integer instead of a string
func (t TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(int32(t))
}

// UnmarshalJSON implements json.Unmarshaler for TaskStatus
// It accepts both integer and string representations
func (t *TaskStatus) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as integer first
	var intVal int32
	if err := json.Unmarshal(data, &intVal); err == nil {
		*t = TaskStatus(intVal)
		return nil
	}

	// Try to unmarshal as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return fmt.Errorf("TaskStatus must be an integer or string, got: %s", string(data))
	}

	// Parse string value
	switch strVal {
	case "TASK_STATUS_UNSPECIFIED":
		*t = TaskStatus_TASK_STATUS_UNSPECIFIED
	case "TASK_STATUS_PENDING":
		*t = TaskStatus_TASK_STATUS_PENDING
	case "TASK_STATUS_RUNNING":
		*t = TaskStatus_TASK_STATUS_RUNNING
	case "TASK_STATUS_COMPLETED":
		*t = TaskStatus_TASK_STATUS_COMPLETED
	case "TASK_STATUS_FAILED":
		*t = TaskStatus_TASK_STATUS_FAILED
	case "TASK_STATUS_CANCELLED":
		*t = TaskStatus_TASK_STATUS_CANCELLED
	default:
		return fmt.Errorf("unknown TaskStatus: %s", strVal)
	}
	return nil
}
