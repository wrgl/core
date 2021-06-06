// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package merge

import (
	"github.com/wrgl/core/pkg/objects"
)

type Merge struct {
	ColDiff     *objects.ColDiff
	PK          []byte
	Base        []byte
	Others      [][]byte
	ResolvedRow []string
	Resolved    bool
}
