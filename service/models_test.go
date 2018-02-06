package main

import (
	"testing"
)

func TestSourceAndDestinationID(t *testing.T) {
	tables := []struct {
		ID     uint
		srcID  uint
		destID uint

		source      uint
		destination uint
	}{
		{1, 2, 0, 2, 1},
		{1, 0, 3, 1, 3},
		{2, 0, 0, 0, 0},
		{2, 4, 5, 4, 2}, // doesn't check for an ambiguity
	}

	for _, table := range tables {
		transfer := &Payment{AccountID: table.ID, AccountToID: table.destID, AccountFromID: table.srcID}
		srcID, destID := transfer.SourceID(), transfer.DestinationID()
		if srcID != table.source || destID != table.destination {
			t.Errorf("Wrong source/destination IDs in %s", transfer)
		}
	}
}

func TestPaymentInversion(t *testing.T) {
	payments := []struct {
		source   Payment
		expected Payment
	}{
		{Payment{AccountID: 1, AccountFromID: 2}, Payment{AccountID: 2, AccountToID: 1, AccountFromID: 0}},
		{Payment{AccountID: 1, AccountToID: 2}, Payment{AccountID: 2, AccountFromID: 1, AccountToID: 0}},
	}

	for _, test := range payments {
		inverse := test.source.InversePayment()
		if inverse != test.expected {
			t.Errorf("Expected %v, got %v", test.expected, inverse)
		}
	}
}
