package services

import (
	"strings"
	"testing"
)

func TestParseTeacherLeaveCommands(t *testing.T) {
	tests := []struct {
		input       string
		expectedCmd string
		expectedCode string
		expectedReason string
	}{
		{"acc IZIN-123456", "approve_leave", "IZIN-123456", ""},
		{"ACC IZIN-ABCD89", "approve_leave", "IZIN-ABCD89", ""},
		{"setujui izin-xyz789", "approve_leave", "IZIN-XYZ789", ""},
		{"acc `IZIN-TEST1`", "approve_leave", "IZIN-TEST1", ""},
		{"acc *IZIN-TEST2*", "approve_leave", "IZIN-TEST2", ""},
		{"acc", "approve_leave_help", "", ""},
		{"setujui", "approve_leave_help", "", ""},
		{"tolak IZIN-123", "reject_leave", "IZIN-123", ""},
		{"tolak IZIN-123 Surat dokter tidak ada", "reject_leave", "IZIN-123", "Surat dokter tidak ada"},
		{"tolak", "reject_leave_help", "", ""},
		{"izin pending", "list_pending_leaves", "", ""},
		{"list izin", "list_pending_leaves", "", ""},
		{"cek izin", "list_pending_leaves", "", ""},
		{"daftar izin", "list_pending_leaves", "", ""},
	}

	for _, tt := range tests {
		cmd := ParseTeacherCommand(tt.input)
		if cmd.Command != tt.expectedCmd {
			t.Errorf("For input %q: expected command %q, got %q", tt.input, tt.expectedCmd, cmd.Command)
		}
		if cmd.Code != tt.expectedCode {
			t.Errorf("For input %q: expected code %q, got %q", tt.input, tt.expectedCode, cmd.Code)
		}
		if cmd.Reason != tt.expectedReason {
			t.Errorf("For input %q: expected reason %q, got %q", tt.input, tt.expectedReason, cmd.Reason)
		}
	}
}

func TestFormatDateRangeIndo(t *testing.T) {
	same := FormatDateRangeIndo("2026-11-17", "2026-11-17")
	if same != "17/11/2026" {
		t.Errorf("Expected '17/11/2026', got %q", same)
	}

	diff := FormatDateRangeIndo("2026-11-17", "2026-11-19")
	if diff != "17/11/2026 s/d 19/11/2026" {
		t.Errorf("Expected '17/11/2026 s/d 19/11/2026', got %q", diff)
	}
}

func TestGetUniquePhones(t *testing.T) {
	phones := getUniquePhones("081234567890", "+6281234567890", "081234567890", "628987654321")
	if len(phones) != 2 {
		t.Errorf("Expected 2 unique phones, got %d: %v", len(phones), phones)
	}
}

func TestGenerateParentLeaveNotifications(t *testing.T) {
	approvedMsg := GenerateParentLeaveApprovedNotification("Ahmad", "izin", "2026-11-17", "2026-11-18", "Budi Santoso")
	if !strings.Contains(approvedMsg, "DISETUJUI") || !strings.Contains(approvedMsg, "Budi Santoso") {
		t.Errorf("Approved notification content invalid: %s", approvedMsg)
	}

	rejectedMsg := GenerateParentLeaveRejectedNotification("Ahmad", "izin", "2026-11-17", "2026-11-18", "Budi Santoso", "Surat dokter tidak dilampirkan")
	if !strings.Contains(rejectedMsg, "BELUM DAPAT DISETUJUI") || !strings.Contains(rejectedMsg, "Surat dokter tidak dilampirkan") {
		t.Errorf("Rejected notification content invalid: %s", rejectedMsg)
	}
}
