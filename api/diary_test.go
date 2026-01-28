package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiaryEntries_Unauthenticated_Visibility(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/diary-entries", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DiaryEntries)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var entries []DiaryEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}

	// Check that private entries are returned but censored
	foundPrivate := false
	for _, entry := range entries {
		if entry.IsPrivate {
			foundPrivate = true
			if entry.Content != "Login to unlock" {
				t.Errorf("Private entry content leaked: %v", entry.Content)
			}
			// Check title format "Entry 002 [Private]"
			expectedTitlePrefix := "Entry "
			if len(entry.Title) < len(expectedTitlePrefix) || entry.Title[len(entry.Title)-9:] != "[Private]" {
				t.Errorf("Private entry title not properly censored: %v", entry.Title)
			}
		}
	}
	
	if !foundPrivate {
		// If we are testing with default data, there should be a private entry.
		// If testing with live data that might have no private entries, this warning is acceptable,
		// but for this repro test we assume default data or existing data has at least one private.
		// Given the previous test run had private entries, we expect to find them.
		t.Error("No private entries found in response (they should be present but censored)")
	}
}

func TestDiaryEntries_Authenticated_Visibility(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/diary-entries", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer true")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DiaryEntries)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var entries []DiaryEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}

	// Should find at least one private entry (based on default data)
	foundPrivate := false
	for _, entry := range entries {
		if entry.IsPrivate {
			foundPrivate = true
			break
		}
	}

	if !foundPrivate {
		t.Error("Did not find any private entries in authenticated response (expected at least one default private entry)")
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		num    int
		width  int
		expect string
	}{
		{1, 3, "001"},
		{10, 3, "010"},
		{100, 3, "100"},
		{1000, 3, "1000"},
		{12345, 3, "12345"},
		{0, 3, "000"},
	}

	for _, tt := range tests {
		got := padLeft(tt.num, tt.width)
		if got != tt.expect {
			t.Errorf("padLeft(%d, %d) = %s; want %s", tt.num, tt.width, got, tt.expect)
		}
	}
}
