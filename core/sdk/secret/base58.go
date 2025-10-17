package secret

// Base58 alphabet (Bitcoin-style, no 0/O/I/l ambiguity)
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// EncodeBase58 encodes an 8-byte slice to Base58 string
// Used for compact, readable secret IDs (64-bit random values)
// Input must be exactly 8 bytes
func EncodeBase58(data []byte) string {
	if len(data) != 8 {
		panic("EncodeBase58 requires exactly 8 bytes")
	}
	if len(data) == 0 {
		return ""
	}

	// Convert bytes to big integer (little-endian)
	var num [8]byte
	copy(num[:], data)

	// Convert to base58
	var result []byte
	for i := 0; i < 8; i++ {
		if num[i] == 0 && i == 7 {
			continue
		}

		// Divide by 58
		var remainder byte
		for j := 0; j < 8; j++ {
			temp := int(num[j]) + int(remainder)*256
			num[j] = byte(temp / 58)
			remainder = byte(temp % 58)
		}

		result = append([]byte{base58Alphabet[remainder]}, result...)
	}

	// Handle leading zeros
	for i := 0; i < len(data); i++ {
		if data[i] != 0 {
			break
		}
		result = append([]byte{'1'}, result...)
	}

	return string(result)
}
