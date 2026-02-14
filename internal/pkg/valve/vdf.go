package valve

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// VDFNode represents a node in a VDF (Valve Data Format) tree.
// Value is empty for section nodes; Children are nested sections.
type VDFNode struct {
	Key      string
	Value    string
	Children []*VDFNode
}

// ParseVDF parses full VDF text into a tree structure.
// Handles quoted strings, nested braces, C-style comments, multi-line values, and Unicode.
func ParseVDF(data []byte) (*VDFNode, error) {
	p := &vdfParser{data: data, pos: 0}
	root, err := p.parseRoot()
	if err != nil {
		return nil, err
	}
	return root, nil
}

type vdfParser struct {
	data []byte
	pos  int
}

func (p *vdfParser) curr() byte {
	if p.pos >= len(p.data) {
		return 0
	}
	return p.data[p.pos]
}

func (p *vdfParser) atEnd() bool {
	return p.pos >= len(p.data)
}

func (p *vdfParser) advance() {
	if p.pos < len(p.data) {
		p.pos++
	}
}

func (p *vdfParser) skipWhitespaceAndComments() {
	for !p.atEnd() {
		c := p.curr()
		if c == '/' && p.pos+1 < len(p.data) && p.data[p.pos+1] == '/' {
			p.skipLineComment()
			continue
		}
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.advance()
			continue
		}
		break
	}
}

func (p *vdfParser) skipLineComment() {
	for !p.atEnd() {
		c := p.curr()
		p.advance()
		if c == '\n' || c == '\r' {
			break
		}
	}
}

func (p *vdfParser) readQuotedString() (string, error) {
	if p.curr() != '"' {
		return "", errors.New("expected opening quote")
	}
	p.advance()
	var sb strings.Builder
	for !p.atEnd() {
		c := p.curr()
		if c == '\\' {
			p.advance()
			if p.atEnd() {
				return "", errors.New("unexpected end after backslash")
			}
			next := p.curr()
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte(next)
			}
			p.advance()
			continue
		}
		if c == '"' {
			p.advance()
			return sb.String(), nil
		}
		if c < utf8.RuneSelf {
			sb.WriteByte(c)
			p.advance()
		} else {
			r, size := utf8.DecodeRune(p.data[p.pos:])
			if r == utf8.RuneError {
				return "", errors.New("invalid UTF-8 in string")
			}
			sb.WriteRune(r)
			p.pos += size
		}
	}
	return "", errors.New("unterminated quoted string")
}

func (p *vdfParser) parseRoot() (*VDFNode, error) {
	p.skipWhitespaceAndComments()
	if p.atEnd() {
		return &VDFNode{Key: "", Value: "", Children: nil}, nil
	}
	return p.parseKeyValueOrBlock()
}

func (p *vdfParser) parseKeyValueOrBlock() (*VDFNode, error) {
	key, err := p.readQuotedString()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if p.atEnd() {
		return &VDFNode{Key: key, Value: "", Children: nil}, nil
	}
	c := p.curr()
	if c == '{' {
		p.advance()
		children, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.curr() == '}' {
			p.advance()
		}
		return &VDFNode{Key: key, Value: "", Children: children}, nil
	}
	if c == '"' {
		val, err := p.readQuotedString()
		if err != nil {
			return nil, err
		}
		return &VDFNode{Key: key, Value: val, Children: nil}, nil
	}
	return nil, errors.New("expected '{' or quoted value after key")
}

func (p *vdfParser) parseBlock() ([]*VDFNode, error) {
	var children []*VDFNode
	for {
		p.skipWhitespaceAndComments()
		if p.atEnd() {
			break
		}
		if p.curr() == '}' {
			break
		}
		if p.curr() != '"' {
			return nil, errors.New("expected quoted key in block")
		}
		child, err := p.parseKeyValueOrBlock()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return children, nil
}

// FindChild returns the first child with the given key (case-insensitive).
func (n *VDFNode) FindChild(key string) *VDFNode {
	keyLower := strings.ToLower(key)
	for _, c := range n.Children {
		if strings.ToLower(c.Key) == keyLower {
			return c
		}
	}
	return nil
}

// GetString returns the leaf value of the first child with the given key (case-insensitive).
func (n *VDFNode) GetString(key string) string {
	child := n.FindChild(key)
	if child == nil {
		return ""
	}
	return child.Value
}

// GetInt returns the int value of the first child with the given key.
func (n *VDFNode) GetInt(key string) int {
	s := n.GetString(key)
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(strings.TrimSpace(s))
	return i
}

// GetFloat returns the float64 value of the first child with the given key.
func (n *VDFNode) GetFloat(key string) float64 {
	s := n.GetString(key)
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	// Remove trailing 'f' if present (common in VDF)
	s = strings.TrimSuffix(s, "f")
	s = strings.TrimSuffix(s, "F")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
