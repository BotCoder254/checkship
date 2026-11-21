package domaincheck

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os"
	"strings"

	"math/rand/v2"

	"golang.org/x/net/dns/dnsmessage"
)

const typeCAA dnsmessage.Type = 257

var errNoNameserver = errors.New("no nameserver")

type caaRecord struct {
	tag   string
	value string
}

// Looks up the CAA records that apply to a name: those of the name itself, or of the closest
// parent that has some (RFC 8659), stopping before the top-level domain. Returns no records when
// none applies. Go's resolver can't query CAA, hence this small DNS client.
type caaLookup func(ctx context.Context, name string) ([]caaRecord, error)

// Queries the nameservers of /etc/resolv.conf, the same ones the system resolver uses.
func systemCAALookup(ctx context.Context, name string) ([]caaRecord, error) {
	servers, err := nameservers("/etc/resolv.conf")

	if err != nil {
		return nil, err
	}

	labels := strings.Split(strings.TrimSuffix(name, "."), ".")

	for i := 0; i < len(labels)-1; i++ {
		current := strings.Join(labels[i:], ".")
		records, err := queryCAA(ctx, servers, current)

		if err != nil {
			return nil, err
		}

		if len(records) > 0 {
			return records, nil
		}
	}

	return nil, nil
}

func queryCAA(ctx context.Context, servers []string, name string) ([]caaRecord, error) {
	question, err := dnsmessage.NewName(name + ".")

	if err != nil {
		return nil, err
	}

	id := uint16(rand.Uint32())
	msg := dnsmessage.Message{
		Header:    dnsmessage.Header{ID: id, RecursionDesired: true},
		Questions: []dnsmessage.Question{{Name: question, Type: typeCAA, Class: dnsmessage.ClassINET}},
	}
	packet, err := msg.Pack()

	if err != nil {
		return nil, err
	}

	var lastErr error = errNoNameserver

	for _, server := range servers {
		answer, err := exchange(ctx, server, packet)

		if err != nil {
			lastErr = err
			continue
		}

		return parseCAA(answer, id)
	}

	return nil, lastErr
}

func exchange(ctx context.Context, server string, packet []byte) ([]byte, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "udp", net.JoinHostPort(server, "53"))

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if _, err = conn.Write(packet); err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)

	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

// Reads the CAA records of a DNS answer. A failed lookup (other than "no such name") is an error,
// so an unreachable resolver never reads as "no CAA record".
func parseCAA(answer []byte, id uint16) ([]caaRecord, error) {
	var parser dnsmessage.Parser
	header, err := parser.Start(answer)

	if err != nil {
		return nil, err
	}

	if header.ID != id || !header.Response {
		return nil, errors.New("unexpected dns answer")
	}

	if header.Truncated {
		return nil, errors.New("truncated dns answer")
	}

	if header.RCode != dnsmessage.RCodeSuccess && header.RCode != dnsmessage.RCodeNameError {
		return nil, errors.New("dns error: " + header.RCode.String())
	}

	if err = parser.SkipAllQuestions(); err != nil {
		return nil, err
	}

	var records []caaRecord

	for {
		h, err := parser.AnswerHeader()

		if errors.Is(err, dnsmessage.ErrSectionDone) {
			return records, nil
		}

		if err != nil {
			return nil, err
		}

		if h.Type != typeCAA {
			if err = parser.SkipAnswer(); err != nil {
				return nil, err
			}
			continue
		}

		body, err := parser.UnknownResource()

		if err != nil {
			return nil, err
		}

		if record, ok := decodeCAA(body.Data); ok {
			records = append(records, record)
		}
	}
}

// CAA RDATA: flags (1 byte), tag length (1 byte), tag, value.
func decodeCAA(data []byte) (caaRecord, bool) {
	if len(data) < 2 {
		return caaRecord{}, false
	}

	tagLen := int(data[1])

	if len(data) < 2+tagLen {
		return caaRecord{}, false
	}

	return caaRecord{
		tag:   strings.ToLower(string(data[2 : 2+tagLen])),
		value: string(data[2+tagLen:]),
	}, true
}

// Whether Let's Encrypt may issue a (non wildcard) certificate under these records, and the CAs
// they allow. Without any "issue" record every CA may (RFC 8659 §4.2).
func allowsLetsEncrypt(records []caaRecord) (bool, []string) {
	var issuers []string

	for _, record := range records {
		if record.tag != "issue" {
			continue
		}

		issuer := strings.TrimSpace(strings.SplitN(record.value, ";", 2)[0])
		issuers = append(issuers, issuer)

		if strings.EqualFold(issuer, "letsencrypt.org") {
			return true, nil
		}
	}

	if len(issuers) == 0 {
		return true, nil
	}

	return false, issuers
}

func nameservers(path string) ([]string, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) >= 2 && fields[0] == "nameserver" {
			servers = append(servers, fields[1])
		}
	}

	if len(servers) == 0 {
		return nil, errNoNameserver
	}

	return servers, scanner.Err()
}
