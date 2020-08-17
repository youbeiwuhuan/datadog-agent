package dogstatsd

import (
	"fmt"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/tagger/collectors"
)

func buildTags(tagCount int) []string {
	tags := make([]string, 0, tagCount)
	for i := 0; i < tagCount; i++ {
		tags = append(tags, fmt.Sprintf("tag%d:val%d", i, i))
	}

	return tags
}

// used to store the result and avoid optimizations
var tags []string

func BenchmarkEnrichTags(b *testing.B) {
	originalGetTags := getTags

	for i := 100; i < 1000; i *= 2 {
		b.Run(fmt.Sprintf("%d-tags", i), func(sb *testing.B) {
			baseTags := buildTags(i)
			extraTags := buildTags(i / 2)
			originTagsFunc := func() []string {
				return extraTags
			}
			getTags = func(entity string, cardinality collectors.TagCardinality) ([]string, error) {
				return extraTags, nil
			}
			sb.ResetTimer()

			for n := 0; n < sb.N; n++ {
				tags, _ = enrichTags(baseTags, "hostname", originTagsFunc, true)
			}
		})
	}

	// Revert to original value
	getTags = originalGetTags
}
