package utils

import "strings"

func Split(text string, maxSize int) (head, tail string) {
	if len(text) < maxSize {
		return head, text
	}

	index, i := 0, len(text)

	for {
		i = strings.LastIndex(text[:i], " ")
		if i == -1 || len(text[i:]) > maxSize {
			break
		}

		index = i
	}

	return text[:index], text[index:]
}
