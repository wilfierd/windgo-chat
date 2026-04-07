package utils

import "testing"

// TestDummy là một bài Test mẫu cơ bản để Github Actions chạy được và báo Pass (Xanh).
// Trong thực tế, lúc code thật bạn sẽ thay bằng việc gọi các hàm (từ validation, jwt,...)
// và check xem kết quả hàm đó trả ra có đúng ý bạn không!
func TestDummy(t *testing.T) {
	// Giả lập làm một phép toán
	result := 1 + 1
	expected := 2

	// Kiểm tra tự động
	// Nếu result khác expected, báo đỏ và đánh sập (abort) quy trình CI/CD!
	if result != expected {
		t.Errorf("Fail: Kỳ vọng kết quả là %d, nhưng chương trình lại tính ra %d", expected, result)
	}
}

