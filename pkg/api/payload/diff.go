// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package payload

type ColDiff struct {
	OldPK      []uint32 `json:"oldPK,omitempty"`
	PK         []uint32 `json:"pk,omitempty"`
	OldColumns []string `json:"oldColumns"`
	Columns    []string `json:"columns"`
}

type RowDiff struct {
	PK        *Hex   `json:"pk,omitempty"`
	Sum       *Hex   `json:"sum,omitempty"`
	OldSum    *Hex   `json:"oldSum,omitempty"`
	Offset    uint32 `json:"offset"`
	OldOffset uint32 `json:"oldOffset"`
}

type DiffResponse struct {
	ColDiff *ColDiff   `json:"colDiff"`
	RowDiff []*RowDiff `json:"rowDiff"`
}
