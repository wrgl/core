// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package payload

import "time"

type CreateTransactionResponse struct {
	ID string `json:"id"`
}

type TxBranch struct {
	Name       string `json:"name"`
	CurrentSum string `json:"currentSum"`
	NewSum     string `json:"newSum"`
}

type GetTransactionResponse struct {
	Status   string     `json:"status"`
	Begin    time.Time  `json:"begin"`
	End      time.Time  `json:"end,omitempty"`
	Branches []TxBranch `json:"branches"`
}

type UpdateTransactionRequest struct {
	Discard bool `json:"discard,omitempty"`
	Commit  bool `json:"commit,omitempty"`
}
