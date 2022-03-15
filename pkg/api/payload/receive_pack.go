// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package payload

type Update struct {
	Sum    *Hex   `json:"sum,omitempty"`
	OldSum *Hex   `json:"oldSum,omitempty"`
	ErrMsg string `json:"errMsg,omitempty"`
}

type ReceivePackRequest struct {
	Updates map[string]*Update `json:"updates"`
}

type ReceivePackResponse struct {
	Updates map[string]*Update `json:"updates"`
}
