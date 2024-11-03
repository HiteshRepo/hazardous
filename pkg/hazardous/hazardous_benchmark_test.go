package hazardous

import "testing"

func BenchmarkExtractCommandName(b *testing.B) {
	cmd := createCallExpr(&testing.T{}, "rm -rf file.txt")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractCommandName(cmd)
	}
}

func BenchmarkHasHazardousFlags(b *testing.B) {
	cmd := createCallExpr(&testing.T{}, "rm -rf file.txt")
	hazardousFlags := []string{"-rf", "-fr"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasHazardousFlags(cmd, hazardousFlags)
	}
}
