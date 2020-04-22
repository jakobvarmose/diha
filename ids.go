package main

func distance(a, b string) string {
	c := make([]byte, len(a))
	for i := range c {
		c[i] = a[i] ^ b[i]
	}
	return string(c)
}

func bucketIndex(a string) int {
	for i := range []byte(a) {
		if a[i] == 0 {
			continue
		}
		for j := 0; j < 8; j++ {
			if a[i]&(128>>uint(j)) != 0 {
				return i*8 + j
			}
		}
	}
	return len(a) * 8
}
