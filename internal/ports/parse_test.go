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

func TestParseLsofOutputSamples(t *testing.T) {
	output := `COMMAND   PID USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
node    12345 dries   21u  IPv4 0x12345      0t0  TCP *:3000 (LISTEN)
Google  44444 dries   98u  IPv6 0x22222      0t0  TCP 127.0.0.1:57498->127.0.0.1:8080 (ESTABLISHED)
`

	entries := parseLsofOutput(output)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	listener := entries[0]
	if listener.Process != "node" || listener.PID != 12345 || listener.Port != 3000 {
		t.Fatalf("unexpected listener entry: %#v", listener)
	}
	if listener.Kind != KindListener || listener.State != "LISTEN" || listener.LocalAddress != "*" {
		t.Fatalf("unexpected listener fields: %#v", listener)
	}

	conn := entries[1]
	if conn.Kind != KindConnection || conn.RemotePort != 8080 || conn.LocalAddress != "127.0.0.1" {
		t.Fatalf("unexpected connection entry: %#v", conn)
	}
}

func TestParseSSOutputSamples(t *testing.T) {
	output := `tcp LISTEN 0 4096 127.0.0.1:8080 0.0.0.0:* users:(("node",pid=1234,fd=22))
udp UNCONN 0 0 0.0.0.0:5353 0.0.0.0:* users:(("avahi-daemon",pid=888,fd=12))
tcp ESTAB 0 0 127.0.0.1:57498 127.0.0.1:8080 users:(("curl",pid=777,fd=5))
`

	entries := parseSSOutput(output)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Process != "node" || entries[0].Kind != KindListener || entries[0].Port != 8080 {
		t.Fatalf("unexpected first entry: %#v", entries[0])
	}
	if entries[1].Protocol != "UDP" || entries[1].Kind != KindListener {
		t.Fatalf("unexpected udp entry: %#v", entries[1])
	}
	if entries[2].Kind != KindConnection || entries[2].RemotePort != 8080 || entries[2].PID != 777 {
		t.Fatalf("unexpected established entry: %#v", entries[2])
	}
}

func TestParseTasklistOutputSamples(t *testing.T) {
	output := `"node.exe","1234","Console","1","24,000 K"
"chrome.exe","777","Console","1","150,000 K"
`

	names, err := parseTasklistOutput(output)
	if err != nil {
		t.Fatalf("unexpected tasklist parse error: %v", err)
	}
	if names[1234] != "node.exe" || names[777] != "chrome.exe" {
		t.Fatalf("unexpected names: %#v", names)
	}
}

func TestParseWindowsNetstatOutputSamples(t *testing.T) {
	pidsToNames := map[int]string{1234: "node.exe", 777: "chrome.exe", 888: "dns.exe"}
	tcpOutput := `  TCP    127.0.0.1:8080     0.0.0.0:0       LISTENING       1234
  TCP    127.0.0.1:57498    127.0.0.1:8080  ESTABLISHED     777
`
	udpOutput := `  UDP    0.0.0.0:5353      *:*                                888
`

	tcpEntries := parseWindowsNetstatOutput("tcp", tcpOutput, pidsToNames)
	udpEntries := parseWindowsNetstatOutput("udp", udpOutput, pidsToNames)
	if len(tcpEntries) != 2 || len(udpEntries) != 1 {
		t.Fatalf("unexpected entry counts: tcp=%d udp=%d", len(tcpEntries), len(udpEntries))
	}
	if tcpEntries[0].Process != "node.exe" || tcpEntries[0].Kind != KindListener {
		t.Fatalf("unexpected tcp listener: %#v", tcpEntries[0])
	}
	if tcpEntries[1].Kind != KindConnection || tcpEntries[1].RemotePort != 8080 {
		t.Fatalf("unexpected tcp connection: %#v", tcpEntries[1])
	}
	if udpEntries[0].Protocol != "UDP" || udpEntries[0].Process != "dns.exe" || udpEntries[0].Kind != KindListener {
		t.Fatalf("unexpected udp entry: %#v", udpEntries[0])
	}
}
