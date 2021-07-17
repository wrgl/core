// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package objects

type Diff struct {
	PK        []byte
	Sum       []byte
	OldSum    []byte
	Offset    uint32
	OldOffset uint32
}
