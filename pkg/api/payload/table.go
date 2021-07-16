// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package payload

type GetTableResponse struct {
	Columns   []string `json:"columns"`
	PK        []uint32 `json:"pk,omitempty"`
	RowsCount uint32   `json:"rowsCount"`
}
