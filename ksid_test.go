// Copyright 2026 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ksid

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestID tests all ID type methods using table-driven tests.
func TestID(t *testing.T) {
	t.Run("NewID", func(t *testing.T) {
		t.Run("generates non-zero ID", func(t *testing.T) {
			id := NewID()
			if id == 0 {
				t.Error("NewID returned zero")
			}
		})

		t.Run("monotonic ordering", func(t *testing.T) {
			ids := make([]ID, 1000)
			for i := range ids {
				ids[i] = NewID()
			}

			for i := 1; i < len(ids); i++ {
				if ids[i] <= ids[i-1] {
					t.Errorf("ID[%d] = %d <= ID[%d] = %d (not monotonic)", i, ids[i], i-1, ids[i-1])
				}
			}
		})

		t.Run("slice increment within same interval", func(t *testing.T) {
			// Generate multiple IDs quickly to trigger same-interval logic
			id1 := NewID()
			id2 := NewID()
			id3 := NewID()

			// All should be unique and ordered
			if id1 >= id2 || id2 >= id3 {
				t.Error("IDs not properly ordered within same 10µs interval")
			}
		})

		t.Run("slice overflow", func(t *testing.T) {
			// Save and restore state
			idMu.Lock()
			oldInstance := idInstance
			oldTotal := idTotalInstances
			oldLastT := idLastT10us
			idMu.Unlock()
			t.Cleanup(func() {
				idMu.Lock()
				idInstance = oldInstance
				idTotalInstances = oldTotal
				idLastT10us = oldLastT
				idMu.Unlock()
			})

			// Set up to overflow immediately
			// Since we are in the same package, we can manipulate internal state.
			idMu.Lock()
			idLastT10us = max(0, time.Now().UnixMicro()/10-epoch)
			idSlice = sliceMask
			idTotalInstances = 1
			idMu.Unlock()

			// This call should trigger overflow if it happens in the same 10µs interval.
			// If not, we'll try again.
			for range 1000 {
				id := NewID()
				// If we triggered overflow, it should have waited for a new interval
				// or at least reset the slice.
				// Since we set idSlice = sliceMask and idTotalInstances = 1,
				// if NewID() returns an ID with Slice() != sliceMask, it means
				// it either went to a new interval or it overflowed and waited.
				// Actually, if it overflowed and waited, it will definitely have a new interval.
				if id.Slice() == 0 {
					return
				}

				// Reset for next attempt if it happened to be a new interval already
				idMu.Lock()
				idLastT10us = max(0, time.Now().UnixMicro()/10-epoch)
				idSlice = sliceMask
				idMu.Unlock()
			}
			t.Fatal("could not trigger same-interval IDs")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name      string
				id        ID
				wantLen   int    // -1 means check <= idEncodedLen
				wantExact string // empty means check is non-empty
			}{
				{"zero ID returns zero char", 0, 1, "0"},
				{"small ID has compact encoding", 1, -1, ""},
				{"max ID has full length", ID(^uint64(0) >> 1), idEncodedLen, ""}, // Max positive value
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got := tt.id.String()
					if tt.wantExact != "" && got != tt.wantExact {
						t.Errorf("String() = %q, want %q", got, tt.wantExact)
					}
					if tt.wantLen > 0 && len(got) != tt.wantLen {
						t.Errorf("String() length = %d, want %d", len(got), tt.wantLen)
					}
					if tt.wantLen == -1 && len(got) > idEncodedLen {
						t.Errorf("String() length = %d, want <= %d", len(got), idEncodedLen)
					}
				})
			}
		})

		t.Run("lexicographic order", func(t *testing.T) {
			ids := make([]ID, 100)
			for i := range ids {
				ids[i] = NewID()
			}

			encoded := make([]string, len(ids))
			for i, id := range ids {
				encoded[i] = id.String()
			}

			sortedEncoded := make([]string, len(encoded))
			copy(sortedEncoded, encoded)
			sort.Strings(sortedEncoded)

			for i := range encoded {
				if encoded[i] != sortedEncoded[i] {
					t.Errorf("Lexicographic order mismatch at %d: got %s, want %s", i, sortedEncoded[i], encoded[i])
				}
			}
		})
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name string
				id   ID
				want string
			}{
				{"zero ID", 0, `"0"`},
				{"small ID", 1, `"2"`},        // ID 1 encodes to "2" in base32 hex
				{"generated ID", NewID(), ""}, // Will verify round-trip
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := tt.id.MarshalJSON()
					if err != nil {
						t.Fatalf("MarshalJSON error: %v", err)
					}
					if tt.want != "" && string(got) != tt.want {
						t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
					}
					// Verify it's valid JSON string
					var s string
					if err := json.Unmarshal(got, &s); err != nil {
						t.Errorf("MarshalJSON produced invalid JSON: %v", err)
					}
				})
			}
		})

		t.Run("round trip", func(t *testing.T) {
			original := NewID()
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			var decoded ID
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if decoded != original {
				t.Errorf("Round trip failed: got %d, want %d", decoded, original)
			}
		})
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  ID
			}{
				{"zero ID zero char", `"0"`, 0},
				{"zero ID empty", `""`, 0},
				{"small ID", `"2"`, 1}, // "2" decodes to ID 1 in base32 hex
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					var got ID
					if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
						t.Fatalf("UnmarshalJSON error: %v", err)
					}
					if got != tt.want {
						t.Errorf("UnmarshalJSON() = %d, want %d", got, tt.want)
					}
				})
			}
		})

		t.Run("errors", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
			}{
				{"JSON number instead of string", `123`},     // Valid JSON but wrong type
				{"JSON object instead of string", `{"a":1}`}, // Valid JSON but wrong type
				{"invalid ID too long", `"ABCDEFGHIJKLMNO"`}, // 15 chars > 13 max
				{"invalid ID character", `"!!!"`},
				{"invalid high byte character", `"` + string([]byte{200}) + `"`},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					var id ID
					if err := json.Unmarshal([]byte(tt.input), &id); err == nil {
						t.Error("expected error, got nil")
					}
				})
			}
		})
	})

	t.Run("IsZero", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name string
				id   ID
				want bool
			}{
				{"zero ID", 0, true},
				{"non-zero ID", 1, false},
				{"generated ID", NewID(), false},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					if got := tt.id.IsZero(); got != tt.want {
						t.Errorf("IsZero() = %v, want %v", got, tt.want)
					}
				})
			}
		})
	})

	t.Run("Parse", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  ID
			}{
				{"empty string", "", 0},
				{"zero char", "0", 0},
				{"small ID encoded", "2", 1}, // "2" decodes to ID 1 in base32 hex
				{"lowercase supported", "v", 15},
				{"mixed case supported", "1a2b", 21541},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := Parse(tt.input)
					if err != nil {
						t.Fatalf("Parse error: %v", err)
					}
					if got != tt.want {
						t.Errorf("Parse() = %d, want %d", got, tt.want)
					}
				})
			}
		})

		t.Run("round trip", func(t *testing.T) {
			// Test round-trip for various IDs
			ids := []ID{0, 1, 100, NewID(), ID(1 << 30), ID(1 << 60)}
			for _, original := range ids {
				encoded := original.String()
				decoded, err := Parse(encoded)
				if err != nil {
					t.Fatalf("Parse(%q) error: %v", encoded, err)
				}
				if decoded != original {
					t.Errorf("Round trip failed for %d: encoded=%q, decoded=%d", original, encoded, decoded)
				}
			}
		})

		t.Run("errors", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
			}{
				{"too long", "ABCDEFGHIJKLMNO"}, // 15 chars > 13 max
				{"invalid char low", "!!!"},
				{"invalid char dash", "-"},
				{"invalid char high", string([]byte{200})},
				{"mixed valid invalid", "ABC!DEF"},
				{"invalid char W", "W"},
				{"invalid char Z", "Z"},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					_, err := Parse(tt.input)
					if err == nil {
						t.Error("expected error, got nil")
					}
				})
			}
		})
	})

	t.Run("Time", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			before := time.Now()
			id := NewID()
			after := time.Now()

			idTime := id.Time()
			// Truncate to 10µs resolution
			if idTime.Before(before.Truncate(10 * time.Microsecond)) {
				t.Errorf("ID time %v is before creation time %v", idTime, before)
			}
			if idTime.After(after.Add(10 * time.Microsecond)) {
				t.Errorf("ID time %v is after creation time %v", idTime, after)
			}
		})

		t.Run("zero ID time", func(t *testing.T) {
			id := ID(0)
			idTime := id.Time()
			// Zero ID should return epoch time (2026-01-01 00:00:00 UTC)
			expectedEpoch := time.UnixMicro(epoch * 10)
			if !idTime.Equal(expectedEpoch) {
				t.Errorf("Zero ID time = %v, want epoch %v", idTime, expectedEpoch)
			}
		})
	})

	t.Run("Slice", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name string
				id   ID
				want int
			}{
				{"zero ID", 0, 0},
				{"constructed ID slice 0", newIDFromParts(0, 0), 0},
				{"constructed ID slice 1234", newIDFromParts(0, 1234), 1234},
				{"constructed ID slice max", newIDFromParts(0, sliceMask), sliceMask},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					if got := tt.id.Slice(); got != tt.want {
						t.Errorf("Slice() = %d, want %d", got, tt.want)
					}
				})
			}
		})

		t.Run("slice increments within interval", func(t *testing.T) {
			// Generate several IDs quickly and verify slice increments
			id1 := NewID()
			id2 := NewID()
			// If in same interval, slice should increment
			if id1.Time().Equal(id2.Time()) && id2.Slice() != id1.Slice()+1 {
				t.Errorf("Slice not incrementing: id1.Slice()=%d, id2.Slice()=%d", id1.Slice(), id2.Slice())
			}
		})
	})

	t.Run("Compare", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			id1 := ID(100)
			id2 := ID(200)
			id3 := ID(100)

			tests := []struct {
				name string
				a, b ID
				want int
			}{
				{"equal", id1, id3, 0},
				{"less than", id1, id2, -1},
				{"greater than", id2, id1, 1},
				{"zero compare", ID(0), ID(0), 0},
				{"zero less than", ID(0), ID(1), -1},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					if got := tt.a.Compare(tt.b); got != tt.want {
						t.Errorf("Compare() = %d, want %d", got, tt.want)
					}
				})
			}
		})

		t.Run("ordering with generated IDs", func(t *testing.T) {
			id1 := NewID()
			id2 := NewID()

			if id1.Compare(id1) != 0 { //nolint:gocritic // Intentional self-comparison test
				t.Error("Compare(self) != 0")
			}
			if id1.Compare(id2) >= 0 {
				t.Error("Earlier ID should be less than later ID")
			}
			if id2.Compare(id1) <= 0 {
				t.Error("Later ID should be greater than earlier ID")
			}
		})
	})
}

func TestInitIDSlice(t *testing.T) {
	// Reset to default after test
	t.Cleanup(func() {
		idMu.Lock()
		idInstance = 0
		idTotalInstances = 1
		idMu.Unlock()
	})

	t.Run("valid configurations", func(t *testing.T) {
		tests := []struct {
			name           string
			instance       int
			totalInstances int
		}{
			{"single instance", 0, 1},
			{"two instances first", 0, 2},
			{"two instances second", 1, 2},
			{"three instances", 2, 3},
			{"max instances", sliceMask, sliceMask + 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := InitIDSlice(tt.instance, tt.totalInstances); err != nil {
					t.Errorf("InitIDSlice(%d, %d) error: %v", tt.instance, tt.totalInstances, err)
				}
			})
		}
	})

	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			name           string
			instance       int
			totalInstances int
		}{
			{"negative instance", -1, 3},
			{"instance >= total", 3, 3},
			{"zero total", 0, 0},
			{"negative total", 0, -1},
			{"total exceeds max", 0, sliceMask + 2},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := InitIDSlice(tt.instance, tt.totalInstances); err == nil {
					t.Errorf("InitIDSlice(%d, %d) expected error, got nil", tt.instance, tt.totalInstances)
				}
			})
		}
	})

	t.Run("partitions slice space", func(t *testing.T) {
		// Reset state
		idMu.Lock()
		idLastT10us = 0
		idMu.Unlock()

		// Configure 4 instances
		if err := InitIDSlice(2, 4); err != nil {
			t.Fatalf("InitIDSlice error: %v", err)
		}

		// Generate IDs and verify slice values
		id1 := NewID()
		id2 := NewID()
		id3 := NewID()

		// Instance 2 should generate slices: 2, 6, 10, ...
		if id1.Slice()%4 != 2 {
			t.Errorf("id1.Slice() = %d, want %%4 == 2", id1.Slice())
		}
		if id2.Slice()%4 != 2 {
			t.Errorf("id2.Slice() = %d, want %%4 == 2", id2.Slice())
		}
		if id3.Slice()%4 != 2 {
			t.Errorf("id3.Slice() = %d, want %%4 == 2", id3.Slice())
		}
	})
}

func TestIDMarshalText(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name string
			id   ID
			want string
		}{
			{"zero ID", 0, "0"},
			{"small ID", 1, "2"},
			{"generated ID", NewID(), ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.id.MarshalText()
				if err != nil {
					t.Fatalf("MarshalText error: %v", err)
				}
				if tt.want != "" && string(got) != tt.want {
					t.Errorf("MarshalText() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("round trip", func(t *testing.T) {
		original := NewID()
		data, err := original.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText error: %v", err)
		}
		var decoded ID
		if err := decoded.UnmarshalText(data); err != nil {
			t.Fatalf("UnmarshalText error: %v", err)
		}
		if decoded != original {
			t.Errorf("Round trip failed: got %d, want %d", decoded, original)
		}
	})

	t.Run("map key JSON encoding", func(t *testing.T) {
		id1 := NewID()
		id2 := NewID()

		m := map[ID]string{
			id1: "first",
			id2: "second",
		}

		data, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal map error: %v", err)
		}

		var decoded map[ID]string
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal map error: %v", err)
		}

		if decoded[id1] != "first" {
			t.Errorf("decoded[id1] = %q, want %q", decoded[id1], "first")
		}
		if decoded[id2] != "second" {
			t.Errorf("decoded[id2] = %q, want %q", decoded[id2], "second")
		}
	})
}

func TestIDUnmarshalText(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  ID
		}{
			{"zero char", "0", 0},
			{"empty string", "", 0},
			{"small ID", "2", 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var got ID
				if err := got.UnmarshalText([]byte(tt.input)); err != nil {
					t.Fatalf("UnmarshalText error: %v", err)
				}
				if got != tt.want {
					t.Errorf("UnmarshalText() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{"invalid ID too long", "ABCDEFGHIJKLMNO"},
			{"invalid character", "!!!"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var id ID
				if err := id.UnmarshalText([]byte(tt.input)); err == nil {
					t.Error("expected error, got nil")
				}
			})
		}
	})
}

func TestIDList(t *testing.T) {
	t.Run("UnmarshalText", func(t *testing.T) {
		id1 := NewID()
		id2 := NewID()
		id3 := NewID()

		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				want  IDList
			}{
				{"empty string", "", nil},
				{"single ID", id1.String(), IDList{id1}},
				{"multiple IDs", id1.String() + "," + id2.String() + "," + id3.String(), IDList{id1, id2, id3}},
				{"with spaces", id1.String() + " , " + id2.String(), IDList{id1, id2}},
				{"trailing comma", id1.String() + ",", IDList{id1}},
				{"leading comma", "," + id1.String(), IDList{id1}},
				{"empty parts skipped", id1.String() + ",," + id2.String(), IDList{id1, id2}},
				{"zero IDs skipped", id1.String() + ",0," + id2.String(), IDList{id1, id2}},
				{"lowercase supported", strings.ToLower(id1.String()) + "," + id2.String(), IDList{id1, id2}},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					var got IDList
					if err := got.UnmarshalText([]byte(tt.input)); err != nil {
						t.Fatalf("UnmarshalText error: %v", err)
					}
					if len(got) != len(tt.want) {
						t.Fatalf("UnmarshalText() len = %d, want %d", len(got), len(tt.want))
					}
					for i := range got {
						if got[i] != tt.want[i] {
							t.Errorf("UnmarshalText()[%d] = %v, want %v", i, got[i], tt.want[i])
						}
					}
				})
			}
		})

		t.Run("errors", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
			}{
				{"invalid ID", "ABC,invalid!!!,DEF"},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					var list IDList
					if err := list.UnmarshalText([]byte(tt.input)); err == nil {
						t.Error("expected error, got nil")
					}
				})
			}
		})
	})

	t.Run("MarshalText", func(t *testing.T) {
		id1 := NewID()
		id2 := NewID()
		id3 := NewID()

		t.Run("valid", func(t *testing.T) {
			tests := []struct {
				name string
				list IDList
				want string
			}{
				{"nil list", nil, ""},
				{"empty list", IDList{}, ""},
				{"single ID", IDList{id1}, id1.String()},
				{"multiple IDs", IDList{id1, id2, id3}, id1.String() + "," + id2.String() + "," + id3.String()},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := tt.list.MarshalText()
					if err != nil {
						t.Fatalf("MarshalText error: %v", err)
					}
					if string(got) != tt.want {
						t.Errorf("MarshalText() = %q, want %q", got, tt.want)
					}
				})
			}
		})

		t.Run("round trip", func(t *testing.T) {
			original := IDList{id1, id2, id3}
			data, err := original.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText error: %v", err)
			}
			var decoded IDList
			if err := decoded.UnmarshalText(data); err != nil {
				t.Fatalf("UnmarshalText error: %v", err)
			}
			if len(decoded) != len(original) {
				t.Fatalf("Round trip len = %d, want %d", len(decoded), len(original))
			}
			for i := range decoded {
				if decoded[i] != original[i] {
					t.Errorf("Round trip[%d] = %v, want %v", i, decoded[i], original[i])
				}
			}
		})
	})
}

func BenchmarkID(b *testing.B) {
	b.ReportAllocs()
	b.Run("ksid-gen", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			NewID()
		}
	})

	b.Run("uuidv4-gen", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = uuid.NewRandom()
		}
	})

	b.Run("uuidv7-gen", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = uuid.NewV7()
		}
	})

	b.Run("ksid-encode", func(b *testing.B) {
		id := NewID()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = id.String()
		}
	})

	b.Run("uuid-encode", func(b *testing.B) {
		u := uuid.New()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = u.String()
		}
	})

	b.Run("ksid-decode", func(b *testing.B) {
		id := NewID()
		s := id.String()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = Parse(s)
		}
	})

	b.Run("uuid-decode", func(b *testing.B) {
		u := uuid.New()
		s := u.String()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_, _ = uuid.Parse(s)
		}
	})
}
