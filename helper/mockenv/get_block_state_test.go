package mockenv

import "testing"

func TestGetBlocks(t *testing.T) {
	blocks := getTransactions(18500000)
	if len(blocks) != 1000 {
		t.Errorf("Expected 1000 blocks, got %d", len(blocks))
	}
}

func TestGetMainnet(t *testing.T) {
	blocks, headers, states, _, dbTxs := GetMainnetData(t, 18500000)
	if len(blocks) != 1000 {
		t.Errorf("Expected 1000 blocks, got %d", len(blocks))
	}
	if len(states) != 1000 {
		t.Errorf("Expected 1000 states, got %d", len(states))
	}
	if len(dbTxs) != 1000 {
		t.Errorf("Expected 1000 dbTxs, got %d", len(dbTxs))
	}
	if len(headers) != 20000 {
		t.Errorf("Expected 20000 headers, got %d", len(headers))
	}
	for _, dbTx := range dbTxs {
		if dbTx == nil {
			t.Errorf("Expected non-nil dbTx")
		}
		dbTx.Rollback()
	}
}

func TestGetHeader(t *testing.T) {
	headers := getHeaders()
	if len(headers) != 20000 {
		t.Errorf("Expected 20000 headers, got %d", len(headers))
	}
}
