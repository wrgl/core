// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package objects

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func AssertCommitEqual(t *testing.T, a, b *Commit) {
	t.Helper()
	require.Equal(t, a.Table, b.Table)
	require.Equal(t, a.AuthorName, b.AuthorName)
	require.Equal(t, a.AuthorEmail, b.AuthorEmail)
	require.Equal(t, a.Message, b.Message)
	require.Equal(t, a.Parents, b.Parents)
	require.Equal(t, a.Time.Unix(), b.Time.Unix())
	require.Equal(t, a.Time.Format("-0700"), b.Time.Format("-0700"))
}

func AssertCommitsEqual(t *testing.T, sla, slb []*Commit, ignoreOrder bool) {
	t.Helper()
	require.Equal(t, len(sla), len(slb))
	if ignoreOrder {
		sortedCopy := func(obj []*Commit) []*Commit {
			sl := make([]*Commit, len(obj))
			copy(sl, obj)
			sort.Slice(sl, func(i, j int) bool {
				a, b := sl[i], sl[j]
				if string(a.Table) != string(b.Table) {
					return string(a.Table) < string(b.Table)
				}
				if a.AuthorName != b.AuthorName {
					return a.AuthorName < b.AuthorName
				}
				if a.AuthorEmail != b.AuthorEmail {
					return a.AuthorEmail < b.AuthorEmail
				}
				if a.Message != b.Message {
					return a.Message < b.Message
				}
				if !a.Time.Equal(b.Time) {
					return a.Time.Before(b.Time)
				}
				if len(a.Parents) != len(b.Parents) {
					return len(a.Parents) < len(b.Parents)
				}
				for k, p := range a.Parents {
					if string(p) != string(b.Parents[k]) {
						return string(p) < string(b.Parents[k])
					}
				}
				return false
			})
			return sl
		}
		sla = sortedCopy(sla)
		slb = sortedCopy(slb)
	}
	for i, a := range sla {
		AssertCommitEqual(t, a, slb[i])
	}
}
