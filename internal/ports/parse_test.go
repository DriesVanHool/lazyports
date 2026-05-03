package ports

import "testing"

func TestParseLsofNameListener(t *testing.T) {
	localAddr, remoteAddr, localPort, remotePort, kind, ok := parseLsofName("*:8080")
	if !ok {
		t.Fatal("expected listener parse to succeed")
	}
	if kind != KindListener || localPort != 8080 || remotePort != 0 {
		t.Fatalf("unexpected listener parse result: kind=%s local=%d remote=%d", kind, localPort, remotePort)
	}
	if localAddr != "*" || remoteAddr != "" {
		t.Fatalf("unexpected addresses: local=%q remote=%q", localAddr, remoteAddr)
	}
}

func TestParseLsofNameConnectionUsesLocalPort(t *testing.T) {
	localAddr, remoteAddr, localPort, remotePort, kind, ok := parseLsofName("127.0.0.1:57498->127.0.0.1:8080")
	if !ok {
		t.Fatal("expected connection parse to succeed")
	}
	if kind != KindConnection {
		t.Fatalf("unexpected kind: %s", kind)
	}
	if localPort != 57498 || remotePort != 8080 {
		t.Fatalf("unexpected ports: local=%d remote=%d", localPort, remotePort)
	}
	if localAddr != "127.0.0.1" || remoteAddr != "127.0.0.1" {
		t.Fatalf("unexpected addresses: local=%q remote=%q", localAddr, remoteAddr)
	}
}

func TestTargetPIDsForPortIgnoresConnections(t *testing.T) {
	entries := []Entry{
		{Port: 8080, PID: 10, Kind: KindConnection},
		{Port: 8080, PID: 11, Kind: KindListener},
		{Port: 8080, PID: 11, Kind: KindListener},
		{Port: 3000, PID: 12, Kind: KindListener},
	}

	targets := targetPIDsForPort(entries, 8080)
	if len(targets) != 1 || targets[0] != 11 {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}
