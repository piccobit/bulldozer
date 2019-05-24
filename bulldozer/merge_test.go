// Copyright 2019 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bulldozer

import (
	"context"
	"testing"

	"bulldozer/pull"
	"bulldozer/pull/pulltest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockMerger struct {
	MergeCount int
	MergeError error

	DeleteCount int
	DeleteError error
}

func (m *MockMerger) Merge(ctx context.Context, pullCtx pull.Context, method MergeMethod, msg CommitMessage) (string, error) {
	m.MergeCount++
	return "deadbeef", m.MergeError
}

func (m *MockMerger) DeleteHead(ctx context.Context, pullCtx pull.Context) error {
	m.DeleteCount++
	return m.DeleteError
}

func TestCalculateCommitTitle(t *testing.T) {
	defaultPullContext := &pulltest.MockPullContext{
		NumberValue: 12,
		TitleValue:  "This is the PR title!",
		CommitsValue: []*pull.Commit{
			{SHA: "f6374a30ec7a3f2dbf35b40ac984b64358ccd246", Message: "The first commit message!"},
			{SHA: "89aec3244253260261351047f0bf6d9b7626c4f6", Message: "The second commit message!"},
			{SHA: "9907911cde43652c51808f79047c98f0d48ae58f", Message: "The third commit message!"},
		},
	}

	tests := map[string]struct {
		PullContext pull.Context
		Strategy    TitleStrategy
		Output      string
	}{
		"pullRequestTitle": {
			PullContext: defaultPullContext,
			Strategy:    PullRequestTitle,
			Output:      "This is the PR title! (#12)",
		},
		"firstCommitTitle": {
			PullContext: defaultPullContext,
			Strategy:    FirstCommitTitle,
			Output:      "The first commit message! (#12)",
		},
		"firstCommitTitleMultiline": {
			PullContext: &pulltest.MockPullContext{
				NumberValue: 12,
				CommitsValue: []*pull.Commit{
					{SHA: "409c973bbaa5e5e6d8cb0b057f2e74398577aaa0", Message: "This is the title\n\nThe message has\nmore lines\n"},
				},
			},
			Strategy: FirstCommitTitle,
			Output:   "This is the title (#12)",
		},
		"githubDefaultTitle": {
			PullContext: defaultPullContext,
			Strategy:    GithubDefaultTitle,
			Output:      "",
		},
	}

	ctx := context.Background()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := calculateCommitTitle(ctx, test.PullContext, SquashOptions{Title: test.Strategy})
			require.NoError(t, err)
			assert.Equal(t, test.Output, output, "calculated title is incorrect")
		})
	}
}

func TestPushRestrictionMerger(t *testing.T) {
	normal := &MockMerger{}
	restricted := &MockMerger{}
	merger := NewPushRestrictionMerger(normal, restricted)

	ctx := context.Background()
	pullCtx := &pulltest.MockPullContext{}

	_, _ = merger.Merge(ctx, pullCtx, SquashAndMerge, CommitMessage{})
	assert.Equal(t, 1, normal.MergeCount, "normal merge was not called")
	assert.Equal(t, 0, restricted.MergeCount, "restricted merge was incorrectly called")

	_ = merger.DeleteHead(ctx, pullCtx)
	assert.Equal(t, 1, normal.DeleteCount, "normal delete was not called")
	assert.Equal(t, 0, restricted.DeleteCount, "restricted delete was incorrectly called")

	pullCtx.PushRestrictionsValue = true

	_, _ = merger.Merge(ctx, pullCtx, SquashAndMerge, CommitMessage{})
	assert.Equal(t, 1, normal.MergeCount, "normal merge was incorrectly called")
	assert.Equal(t, 1, restricted.MergeCount, "restricted merge was not called")

	_ = merger.DeleteHead(ctx, pullCtx)
	assert.Equal(t, 1, normal.DeleteCount, "normal delete was incorrectly called")
	assert.Equal(t, 1, restricted.DeleteCount, "restricted delete was not called")
}
