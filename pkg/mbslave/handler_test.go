package mbslave

import "testing"

func TestHandler(t *testing.T) {
	var h Handler

	t.Run("NewHandler", func(t *testing.T) {
		h = NewHandler()
		if h == nil {
			t.Errorf("NewHandler() returned nil")
		}
	})

	t.Run("WriteSingleRegister", func(t *testing.T) {
		if err := h.WriteSingleRegister(1, 0, 1); err != nil {
			t.Error(err)
		}
	})

	t.Run("ReadHoldingRegisters", func(t *testing.T) {
		results, err := h.ReadHoldingRegisters(1, 0, 1)
		if err != nil {
			t.Error(err)
		}
		if len(results) != 1 {
			t.Errorf("ReadHoldingRegisters() returned %d results, wanted 1", len(results))
		}
		if results[0] != 1 {
			t.Errorf("ReadHoldingRegisters() returned %d, wanted 1", results[0])
		}
	})

	t.Run("WriteMultipleRegisters", func(t *testing.T) {
		if err := h.WriteMultipleRegisters(1, 0, 5, []uint16{100, 200, 300, 400, 500}); err != nil {
			t.Error(err)
		}
	})
}

func BenchmarkHandler(b *testing.B) {
	h := NewHandler()
	b.ResetTimer()

	b.Run("WriteSingleRegister", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := h.WriteSingleRegister(1, 1, 1); err != nil {
				b.Error(err)
			}
		}
	})

	b.ResetTimer()
	b.Run("ReadHoldingRegisters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			results, err := h.ReadHoldingRegisters(1, 1, 1)
			if err != nil {
				b.Error(err)
			}
			if len(results) != 1 {
				b.Errorf("ReadHoldingRegisters() returned %d results, wanted 1", len(results))
			}
			if results[0] != 1 {
				b.Errorf("ReadHoldingRegisters() returned %d, wanted 1", results[0])
			}
		}
	})

	b.ResetTimer()
	b.Run("WriteMultipleRegisters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := h.WriteMultipleRegisters(1, 0, 5, []uint16{100, 200, 300, 400, 500}); err != nil {
				b.Error(err)
			}
		}
	})
}
