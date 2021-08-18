package main

import "testing"

func TestGetPathSkip0(t *testing.T) {
	expected := "/t1/t2"
	got := SkipNPathParams("/t1/t2", 0)
	if got != expected {
		t.Errorf("\nExpected %v\nGot %v", expected, got)
	}
}

func TestSGetPathSkip1(t *testing.T) {
	expected := "/t2"
	got := SkipNPathParams("/t1/t2", 1)
	if got != expected {
		t.Errorf("\nExpected %v\nGot %v", expected, got)
	}
}

func TestDefaultGetPath(t *testing.T) {
	expected := "/"
	got := SkipNPathParams("", 0)
	if got != expected {
		t.Errorf("\nExpected %v\nGot %v", expected, got)
	}
}
