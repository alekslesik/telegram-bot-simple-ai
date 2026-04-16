package ingest

import (
	"regexp"
	"sort"
	"strconv"
)

var captureRe = regexp.MustCompile(`(?i)FireShot Capture\s+(\d+)`)

func extractOrder(name string) (int, bool) {
	m := captureRe.FindStringSubmatch(name)
	if len(m) != 2 {
		return 0, false
	}

	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

func SortInputFiles(files []InputFile) []InputFile {
	sorted := append([]InputFile(nil), files...)

	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]

		leftOrder, leftOK := extractOrder(left.Name)
		rightOrder, rightOK := extractOrder(right.Name)

		switch {
		case leftOK && rightOK:
			if leftOrder != rightOrder {
				return leftOrder < rightOrder
			}
		case leftOK != rightOK:
			return leftOK
		}

		if !left.ModTime.Equal(right.ModTime) {
			return left.ModTime.Before(right.ModTime)
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}

		return left.Path < right.Path
	})

	return sorted
}
