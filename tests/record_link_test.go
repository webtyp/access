package tests

import (
	"testing"

	"webtyp.com/model"
)

func TestRecordLink_LinksMultipleRecordsIdempotently(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)

	if err := m.LinkRecord("u1", "client_directory", "c1"); err != nil {
		t.Fatal(err)
	}
	if err := m.LinkRecord("u1", "client_directory", "c1"); err != nil {
		t.Fatal(err)
	}
	if err := m.LinkRecord("u1", "client_directory", "c2"); err != nil {
		t.Fatal(err)
	}

	got, err := m.LinkedRecords("u1", "client_directory")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "c1" || got[1] != "c2" {
		t.Fatalf("want [c1 c2], got %v", got)
	}
}

func TestRecordLink_IsolatesByUserAndModule(t *testing.T) {
	m, _, _ := newEnv(t, model.AssuranceSingleFactor)

	if err := m.LinkRecord("u1", "client_directory", "c1"); err != nil {
		t.Fatal(err)
	}
	if err := m.LinkRecord("u2", "client_directory", "c2"); err != nil {
		t.Fatal(err)
	}
	if err := m.LinkRecord("u1", "pet_directory", "p1"); err != nil {
		t.Fatal(err)
	}

	got, err := m.LinkedRecords("u1", "client_directory")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "c1" {
		t.Fatalf("want [c1], got %v", got)
	}
}
