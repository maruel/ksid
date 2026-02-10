// Copyright 2026 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ksid_test

import (
	"fmt"

	"github.com/maruel/ksid"
)

func ExampleNewID() {
	id := ksid.NewID()
	fmt.Printf("Generated a %d-bit ID\n", 64)
	// We can't predict the ID, so we just check it's not zero.
	if !id.IsZero() {
		fmt.Println("ID is non-zero")
	}
	// Output:
	// Generated a 64-bit ID
	// ID is non-zero
}

func ExampleParse() {
	// Parse a known ID string
	id, err := ksid.Parse("2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Parsed ID: %d\n", id)
	// Output:
	// Parsed ID: 1
}

func ExampleID_String() {
	id := ksid.ID(1)
	fmt.Printf("ID 1 as string: %s\n", id.String())
	// Output:
	// ID 1 as string: 2
}

func ExampleID_Time() {
	id := ksid.ID(1)
	// The epoch is 2026-01-01 00:00:00 UTC.
	// ID 1 is the very first possible ID at the epoch with slice 1.
	fmt.Printf("ID 1 time: %s\n", id.Time().UTC())
	// Output:
	// ID 1 time: 2026-01-01 00:00:00 +0000 UTC
}

func ExampleInitIDSlice() {
	// In a multi-process environment, call InitIDSlice once at startup.
	// For example, if this is instance 2 of 5:
	err := ksid.InitIDSlice(2, 5)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	id := ksid.NewID()
	// The slice part of the ID will be partitioned.
	if id.Slice()%5 == 2 {
		fmt.Println("ID belongs to instance 2")
	}
	// Output:
	// ID belongs to instance 2
}