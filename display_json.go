package main

import (
	"strconv"
	"strings"
)

func normalizeDisplayColor(color displayColor) displayColor {
	switch color {
	case displayColorAmber,
		displayColorWhite,
		displayColorCyan,
		displayColorGreen,
		displayColorMagenta,
		displayColorRed,
		displayColorYellow,
		displayColorBlue,
		displayColorGrey,
		displayColorKhaki:
		return color
	default:
		return displayColorGreen
	}
}

func frameJSON(f displayFrame) string {
	var sb strings.Builder
	sb.Grow(8000)
	sb.WriteString(`{"Target":"Display","Data":[`)
	for i, c := range f.Cells {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('[')
		sb.WriteString(strconv.Quote(c.Ch))
		sb.WriteString(`,"`)
		sb.WriteString(string(normalizeDisplayColor(c.Color)))
		sb.WriteString(`",`)
		sb.WriteString(strconv.Itoa(c.Size))
		if c.Inverse {
			sb.WriteString(`,1`)
		}
		sb.WriteByte(']')
	}
	sb.WriteString(`]}`)
	return sb.String()
}
