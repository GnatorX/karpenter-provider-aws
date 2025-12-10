/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instancetype

import (
	"testing"
)

// TestInstanceTypeSuffixParsing tests that the instanceTypeScheme regex correctly parses
// all instance types from the fake test data (zz_generated.describe_instance_types.go),
// ensuring the regex works for real AWS instance types.
func TestInstanceTypeSuffixParsing(t *testing.T) {
	// Instance types from pkg/fake/zz_generated.describe_instance_types.go
	// These represent real AWS instance types used in testing
	instanceTypes := []string{
		"c6g.large",
		"dl1.24xlarge",
		"g4ad.16xlarge",
		"g4dn.8xlarge",
		"inf2.24xlarge",
		"inf2.xlarge",
		"m5.large",
		"m5.metal",
		"m5.xlarge",
		"m6idn.32xlarge",
		"m7i-flex.large",
		"p3.8xlarge",
		"t3.large",
		"t4g.medium",
		"t4g.small",
		"t4g.xlarge",
		"trn1.2xlarge",
	}

	for _, instanceType := range instanceTypes {
		t.Run(instanceType, func(t *testing.T) {
			parts := instanceTypeScheme.FindStringSubmatch(instanceType)

			// Regex should always match and return 6 groups:
			// [0]=full match, [1]=category, [2]=high-mem suffix, [3]=generation, [4]=suffixes, [5]=-flex
			if len(parts) != 6 {
				t.Errorf("instanceTypeScheme regex failed to parse %q: got %d parts, expected 6", instanceType, len(parts))
				return
			}

			category := parts[1]
			generation := parts[3]
			suffixes := parts[4]
			flex := parts[5]

			// Category should not be empty
			if category == "" {
				t.Errorf("instanceTypeScheme parsed empty category for %q", instanceType)
			}

			// Generation should not be empty
			if generation == "" {
				t.Errorf("instanceTypeScheme parsed empty generation for %q", instanceType)
			}

			// Verify excluded suffixes are filtered correctly
			var filteredSuffixes []rune
			for _, s := range suffixes {
				if !excludedInstanceTypeSuffixes[s] {
					filteredSuffixes = append(filteredSuffixes, s)
				}
			}

			t.Logf("%s: category=%s, gen=%s, raw_suffix=%q, filtered=%q, flex=%v",
				instanceType, category, generation, suffixes, string(filteredSuffixes), flex == "-flex")
		})
	}
}

// TestInstanceTypeSuffixFiltering tests that excluded suffixes (a, i, g) are properly filtered
func TestInstanceTypeSuffixFiltering(t *testing.T) {
	testCases := []struct {
		instanceType   string
		expectedRaw    string
		expectedFilter string
		expectedFlex   bool
	}{
		// No suffix
		{"m5.large", "", "", false},
		{"t3.large", "", "", false},
		{"p3.8xlarge", "", "", false},
		{"inf2.xlarge", "", "", false},
		{"trn1.2xlarge", "", "", false},

		// Graviton only (g excluded -> empty)
		{"c6g.large", "g", "", false},
		{"t4g.medium", "g", "", false},

		// AMD + NVMe (a excluded, d kept)
		{"g4ad.16xlarge", "ad", "d", false},

		// Graviton + NVMe (g excluded, d kept)
		{"g4dn.8xlarge", "dn", "dn", false},

		// Intel + NVMe + Network (i excluded, d and n kept)
		{"m6idn.32xlarge", "idn", "dn", false},

		// Intel Flex (i excluded, flex detected separately)
		{"m7i-flex.large", "i", "", true},

		// Additional comprehensive test cases
		{"m5d.xlarge", "d", "d", false},           // NVMe only
		{"m5dn.xlarge", "dn", "dn", false},        // NVMe + Network
		{"m5n.xlarge", "n", "n", false},           // Network only
		{"m5zn.xlarge", "zn", "zn", false},        // High-freq + Network
		{"r7iz.large", "iz", "z", false},          // Intel + High-freq (i excluded)
		{"c6gn.xlarge", "gn", "n", false},         // Graviton + Network (g excluded)
		{"x2iedn.xlarge", "iedn", "edn", false},   // Intel + Extra + NVMe + Network
		{"u-24tb1.metal", "", "", false},          // High-memory (special format)
		{"c7i-flex.large", "i", "", true},         // Intel Flex variant
		{"r5b.large", "b", "b", false},            // EBS optimized
	}

	for _, tc := range testCases {
		t.Run(tc.instanceType, func(t *testing.T) {
			parts := instanceTypeScheme.FindStringSubmatch(tc.instanceType)
			if len(parts) != 6 {
				t.Fatalf("regex didn't match %q: got %d parts", tc.instanceType, len(parts))
			}

			rawSuffix := parts[4]
			flex := parts[5] == "-flex"

			if rawSuffix != tc.expectedRaw {
				t.Errorf("raw suffix mismatch for %q: got %q, want %q", tc.instanceType, rawSuffix, tc.expectedRaw)
			}

			var filtered []rune
			for _, s := range rawSuffix {
				if !excludedInstanceTypeSuffixes[s] {
					filtered = append(filtered, s)
				}
			}

			if string(filtered) != tc.expectedFilter {
				t.Errorf("filtered suffix mismatch for %q: got %q, want %q", tc.instanceType, string(filtered), tc.expectedFilter)
			}

			if flex != tc.expectedFlex {
				t.Errorf("flex mismatch for %q: got %v, want %v", tc.instanceType, flex, tc.expectedFlex)
			}
		})
	}
}
