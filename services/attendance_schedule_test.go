package services

import (
	"strings"
	"testing"
)

func TestCalculateTotalSeconds(t *testing.T) {
	tests := []struct {
		name     string
		masuk    string
		pulang   string
		expected int
	}{
		{
			name:     "Normal time difference",
			masuk:    "07:30:00",
			pulang:   "12:00:00",
			expected: 16200, // 4 hours 30 mins = 4.5 * 3600
		},
		{
			name:     "Short time format HH:MM",
			masuk:    "08:00",
			pulang:   "12:00",
			expected: 14400, // 4 hours
		},
		{
			name:     "Pulang earlier than masuk",
			masuk:    "13:00:00",
			pulang:   "12:00:00",
			expected: 0,
		},
		{
			name:     "Empty strings",
			masuk:    "",
			pulang:   "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := calculateTotalSeconds(tt.masuk, tt.pulang)
			if res != tt.expected {
				t.Errorf("calculateTotalSeconds(%q, %q) = %d, expected %d", tt.masuk, tt.pulang, res, tt.expected)
			}
		})
	}
}

func TestGenerateAttendanceConfirmationWithJamPulang(t *testing.T) {
	msg := GenerateAttendanceConfirmation("Budi Santoso", "X IPA 1", "H", "Hadir (dicatat oleh Guru)", "12:00:00")
	if !strings.Contains(msg, "Jam Masuk:") {
		t.Errorf("Expected msg to contain 'Jam Masuk:', got: %s", msg)
	}
	if !strings.Contains(msg, "Jam Pulang: 12:00") {
		t.Errorf("Expected msg to contain 'Jam Pulang: 12:00', got: %s", msg)
	}

	// Status non-Hadir
	msgIzin := GenerateAttendanceConfirmation("Siti", "X IPA 1", "I", "Izin keluarga")
	if strings.Contains(msgIzin, "Jam Pulang:") {
		t.Errorf("Did not expect 'Jam Pulang:' for Izin status, got: %s", msgIzin)
	}
}

func TestGenerateNotificationsWithJamPulang(t *testing.T) {
	studentMsg := GenerateStudentAttendanceNotification("Budi Santoso", "H", "Hadir", "Pak Guru", "12:00:00")
	if !strings.Contains(studentMsg, "Jam Masuk:") || !strings.Contains(studentMsg, "Jam Pulang: 12:00") {
		t.Errorf("Student notification missing Jam Masuk or Jam Pulang: %s", studentMsg)
	}

	parentMsg := GenerateParentAttendanceNotification("Budi Santoso", "X IPA 1", "H", "Hadir", "Pak Guru", "12:00:00")
	if !strings.Contains(parentMsg, "Jam Masuk:") || !strings.Contains(parentMsg, "Jam Pulang: 12:00") {
		t.Errorf("Parent notification missing Jam Masuk or Jam Pulang: %s", parentMsg)
	}
}
