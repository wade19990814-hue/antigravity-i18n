package core

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// asarIntegrityBlockSize is the block size Electron uses when hashing archive
// contents. It has to match @electron/asar or integrity validation fails.
const asarIntegrityBlockSize = 4 * 1024 * 1024

type asarIntegrity struct {
	Algorithm string   `json:"algorithm"`
	Hash      string   `json:"hash"`
	BlockSize int64    `json:"blockSize"`
	Blocks    []string `json:"blocks"`
}

// asarNode mirrors the JSON shape of an asar header entry. Nodes are
// heterogeneous: file nodes carry size/offset/integrity, directory nodes carry
// "files". A hand written checker both parses the tree and normalises an
// offset that arrives as a number instead of a string.
type asarNode struct {
	Size      *int64
	Offset    *string
	Unpacked  *bool
	Executed  bool
	Link      *string
	Integrity *asarIntegrity
	Files     map[string]*asarNode
}

func (n *asarNode) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["size"]; ok {
		var s int64
		if err := json.Unmarshal(v, &s); err == nil {
			n.Size = &s
		}
	}
	if v, ok := raw["offset"]; ok {
		var o string
		if err := json.Unmarshal(v, &o); err == nil {
			n.Offset = &o
		} else {
			var num int64
			if err := json.Unmarshal(v, &num); err == nil {
				s := strconv.FormatInt(num, 10)
				n.Offset = &s
			}
		}
	}
	if v, ok := raw["unpacked"]; ok {
		var u bool
		if err := json.Unmarshal(v, &u); err == nil {
			n.Unpacked = &u
		}
	}
	if v, ok := raw["executed"]; ok {
		_ = json.Unmarshal(v, &n.Executed)
	}
	if v, ok := raw["link"]; ok {
		var l string
		if err := json.Unmarshal(v, &l); err == nil {
			n.Link = &l
		}
	}
	if v, ok := raw["integrity"]; ok {
		var ig asarIntegrity
		if err := json.Unmarshal(v, &ig); err == nil {
			n.Integrity = &ig
		}
	}
	if v, ok := raw["files"]; ok {
		var files map[string]*asarNode
		if err := json.Unmarshal(v, &files); err != nil {
			return err
		}
		n.Files = files
	}
	return nil
}

// span records where a node's size, offset and integrity members sit inside
// the raw header bytes, so values can be replaced without re-serialising the
// document and disturbing the key order Electron wrote.
type span struct {
	offsetV [2]int
	sizeV   [2]int
	integrV [2]int
}

func noSpan() [2]int { return [2]int{-1, -1} }

// Archive is a parsed asar file.
type Archive struct {
	Path       string
	HeaderSize uint32
	Base       int64
	Root       *asarNode

	rawHeader []byte
	spans     map[string]*span
}

// ReadArchive parses the asar header of the file at path.
func ReadArchive(path string) (*Archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// On-disk layout:
	//   [uint32 payloadSize][uint32 headerSize][header pickle][file bodies]
	// The first pickle carries only the header buffer's length; the header
	// pickle itself holds [payloadSize][strLen][JSON][4-byte alignment pad].
	var lead [8]byte
	if _, err := io.ReadFull(f, lead[:]); err != nil {
		return nil, fmt.Errorf("not an asar archive (too short): %w", err)
	}
	headerSize := binary.LittleEndian.Uint32(lead[4:8])
	if headerSize < 16 || int64(headerSize) > 64<<20 {
		return nil, fmt.Errorf("unreasonable asar header size: %d", headerSize)
	}
	if int64(headerSize) > info.Size()-8 {
		return nil, errors.New("asar header extends beyond the archive")
	}

	hdr := make([]byte, headerSize)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return nil, fmt.Errorf("truncated asar header: %w", err)
	}
	if len(hdr) < 8 {
		return nil, errors.New("asar header pickle is truncated")
	}
	jsonLen := binary.LittleEndian.Uint32(hdr[4:8])
	if int64(jsonLen) > int64(len(hdr)-8) {
		return nil, errors.New("asar header JSON length exceeds header size")
	}
	raw := hdr[8 : 8+jsonLen]

	var root asarNode
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("asar header is not valid JSON: %w", err)
	}
	if root.Files == nil {
		return nil, errors.New("asar header has no files map")
	}

	arc := &Archive{
		Path:       path,
		HeaderSize: headerSize,
		Base:       int64(8 + headerSize),
		Root:       &root,
		rawHeader:  raw,
		spans:      indexNodes(raw),
	}
	var validate func(*asarNode, string) error
	validate = func(node *asarNode, rel string) error {
		if node == nil {
			return fmt.Errorf("asar entry is null: %s", rel)
		}
		if node.Files != nil {
			for name, child := range node.Files {
				if err := validate(child, rel+"/"+name); err != nil {
					return err
				}
			}
			return nil
		}
		if node.Link != nil || (node.Unpacked != nil && *node.Unpacked) {
			return nil
		}
		_, err := packedOffset(node, rel, info.Size()-arc.Base)
		return err
	}
	if err := validate(&root, ""); err != nil {
		return nil, err
	}
	return arc, nil
}

// packedOffset checks lengths before allocation or arithmetic on archive data.
func packedOffset(node *asarNode, rel string, payloadSize int64) (int64, error) {
	if node.Size == nil || node.Offset == nil {
		return 0, fmt.Errorf("asar entry is incomplete: %s", rel)
	}
	off, err := strconv.ParseInt(*node.Offset, 10, 64)
	if err != nil || off < 0 || *node.Size < 0 || off > payloadSize || *node.Size > payloadSize-off {
		return 0, fmt.Errorf("asar entry lies outside the archive: %s", rel)
	}
	return off, nil
}

func toSlash(p string) string { return strings.ReplaceAll(p, string(os.PathSeparator), "/") }

// lookup resolves a slash-separated path to a file node, or nil when the path
// is absent or names a directory.
func (a *Archive) lookup(rel string) *asarNode {
	cur := a.Root
	for _, part := range strings.Split(toSlash(rel), "/") {
		if cur.Files == nil {
			return nil
		}
		next, ok := cur.Files[part]
		if !ok || next == nil {
			return nil
		}
		cur = next
	}
	if cur.Files != nil {
		return nil
	}
	return cur
}

// ReadFile returns the bytes of a packed entry.
func (a *Archive) ReadFile(rel string) ([]byte, error) {
	node := a.lookup(rel)
	if node == nil {
		return nil, fmt.Errorf("asar entry not found: %s", rel)
	}
	if node.Unpacked != nil && *node.Unpacked {
		return nil, fmt.Errorf("asar entry is unpacked, read it from app.asar.unpacked: %s", rel)
	}
	f, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	off, err := packedOffset(node, rel, info.Size()-a.Base)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(a.Base+off, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, *node.Size)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, fmt.Errorf("short read for asar entry %s: %w", rel, err)
	}
	return buf, nil
}

// FileEntries lists packed entries in the order their bodies occupy the
// archive, which is the order Electron assigned offsets.
//
// The order comes from the offsets themselves, not from sorting paths: asar's
// offsets reflect the insertion order of a filesystem crawl, which is not the
// same as lexical order. Guessing it wrong would move every body in the rebuilt
// archive.
func (a *Archive) FileEntries() []string {
	type entry struct {
		rel   string
		off   int64
		empty bool
	}
	var entries []entry
	var walk func(n *asarNode, prefix string)
	walk = func(n *asarNode, prefix string) {
		for k, c := range n.Files {
			p := k
			if prefix != "" {
				p = prefix + "/" + k
			}
			if c.Files != nil {
				walk(c, p)
				continue
			}
			if c.Unpacked != nil && *c.Unpacked {
				continue
			}
			if c.Offset == nil {
				continue
			}
			off, err := strconv.ParseInt(*c.Offset, 10, 64)
			if err != nil {
				continue
			}
			entries = append(entries, entry{rel: p, off: off, empty: c.Size != nil && *c.Size == 0})
		}
	}
	walk(a.Root, "")

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].off != entries[j].off {
			return entries[i].off < entries[j].off
		}
		// Empty files share the next body's offset. Keep them before that
		// body so rewriting does not move their offsets past its contents.
		if entries[i].empty != entries[j].empty {
			return entries[i].empty
		}
		return entries[i].rel < entries[j].rel
	})

	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.rel)
	}
	return out
}

// Identity returns the name and version recorded in the archive's package.json
// entry. A missing or malformed package.json yields nil, not a guess.
func (a *Archive) Identity() *Identity {
	data, err := a.ReadFile("package.json")
	if err != nil {
		return nil
	}
	var pkg struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	if pkg.Name == "" || pkg.Version == "" {
		return nil
	}
	return &Identity{Name: pkg.Name, Version: pkg.Version}
}

// integrity computes the asar integrity object for a file's contents: a
// whole-file SHA-256 plus per-4MB block hashes, all lowercase hex.
func integrity(data []byte) *asarIntegrity {
	whole := sha256.Sum256(data)
	var blocks []string
	for start := 0; start < len(data) || start == 0; start += asarIntegrityBlockSize {
		end := start + asarIntegrityBlockSize
		if end > len(data) {
			end = len(data)
		}
		sum := sha256.Sum256(data[start:end])
		blocks = append(blocks, hex.EncodeToString(sum[:]))
		if end >= len(data) {
			break
		}
	}
	return &asarIntegrity{
		Algorithm: "SHA256",
		Hash:      hex.EncodeToString(whole[:]),
		BlockSize: asarIntegrityBlockSize,
		Blocks:    blocks,
	}
}

// Patch is a single file replacement inside an existing archive.
type Patch struct {
	Path string
	Data []byte
}

// plannedBody is one file's place in the archive about to be written.
type plannedBody struct {
	rel       string
	data      []byte
	sourceOff int64
	newOff    int64
	newSize   int64
	changed   bool
}

// WritePatched writes an archive to destPath that is identical to srcPath apart
// from the listed files.
//
// Bodies are never re-serialised: unchanged entries are copied straight from
// the source, and only offsets after the first changed file are recomputed.
// Header bytes are spliced in place so the document's key order and formatting
// survive untouched.
func WritePatched(srcPath, destPath string, patches []Patch) error {
	arc, err := ReadArchive(srcPath)
	if err != nil {
		return err
	}

	repl := make(map[string][]byte, len(patches))
	for _, p := range patches {
		node := arc.lookup(p.Path)
		if node == nil || node.Offset == nil || (node.Unpacked != nil && *node.Unpacked) {
			return fmt.Errorf("cannot patch missing or unpacked asar entry: %s", p.Path)
		}
		repl[toSlash(p.Path)] = p.Data
	}

	var bodies []plannedBody
	offset := int64(0)
	for _, rel := range arc.FileEntries() {
		node := arc.lookup(rel)
		if node == nil {
			return fmt.Errorf("asar entry disappeared while patching: %s", rel)
		}
		var data []byte
		size := *node.Size
		sourceOff, _ := strconv.ParseInt(*node.Offset, 10, 64)
		changed := false
		if d, ok := repl[rel]; ok {
			data = d
			size = int64(len(d))
			changed = true
		}
		bodies = append(bodies, plannedBody{
			rel:       rel,
			data:      data,
			sourceOff: sourceOff,
			newOff:    offset,
			newSize:   size,
			changed:   changed,
		})
		offset += size
	}

	newHeader, err := arc.spliceHeader(bodies)
	if err != nil {
		return err
	}

	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()
	srcInfo, err := in.Stat()
	if err != nil {
		return err
	}
	if dstInfo, err := os.Stat(destPath); err == nil && os.SameFile(srcInfo, dstInfo) {
		return errors.New("source and destination archives must be different files")
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// The size pickle precedes the header pickle. It holds one uint32, so its
	// payload size is 4 and bytes 4-7 carry the header buffer length.
	var sizeBuf [8]byte
	binary.LittleEndian.PutUint32(sizeBuf[0:4], 4)
	binary.LittleEndian.PutUint32(sizeBuf[4:8], uint32(len(newHeader)))
	if _, err := out.Write(sizeBuf[:]); err != nil {
		return err
	}
	if _, err := out.Write(newHeader); err != nil {
		return err
	}
	for _, b := range bodies {
		if b.changed {
			if _, err := out.Write(b.data); err != nil {
				return err
			}
		} else {
			n, err := io.Copy(out, io.NewSectionReader(in, arc.Base+b.sourceOff, b.newSize))
			if err != nil {
				return err
			}
			if n != b.newSize {
				return fmt.Errorf("short read while copying asar entry: %s", b.rel)
			}
		}
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return out.Close()
}

// spliceHeader rewrites the offset, size and integrity members of the raw
// header bytes.
//
// Every edit is expressed as a range in the ORIGINAL document, and the edits
// are applied from the end of the file backwards, so the byte position of a
// later edit has not moved yet when it is applied.
func (a *Archive) spliceHeader(bodies []plannedBody) ([]byte, error) {
	type edit struct {
		start int
		end   int
		text  string
	}
	var edits []edit

	add := func(r [2]int, text string) {
		if r[0] < 0 || r[1] < r[0] {
			return
		}
		edits = append(edits, edit{start: r[0], end: r[1], text: text})
	}

	for _, b := range bodies {
		sp, ok := a.spans[b.rel]
		if !ok {
			return nil, fmt.Errorf("asar header has no span for %s", b.rel)
		}

		// Offsets are JSON strings in asar; keep that representation so the
		// document stays the shape Electron wrote.
		if sp.offsetV[0] >= 0 {
			raw := a.rawHeader[sp.offsetV[0]:sp.offsetV[1]]
			quoted := len(raw) > 0 && raw[0] == byte('"')
			if cur, _ := decodeJSONInt(raw); cur != b.newOff {
				text := strconv.FormatInt(b.newOff, 10)
				if quoted {
					text = strconv.Quote(text)
				}
				add(sp.offsetV, text)
			}
		}

		if sp.sizeV[0] >= 0 {
			raw := a.rawHeader[sp.sizeV[0]:sp.sizeV[1]]
			quoted := len(raw) > 0 && raw[0] == byte('"')
			if cur, _ := decodeJSONInt(raw); cur != b.newSize {
				text := strconv.FormatInt(b.newSize, 10)
				if quoted {
					text = strconv.Quote(text)
				}
				add(sp.sizeV, text)
			}
		}

		if !b.changed {
			// Untouched content keeps the hashes Electron computed when it
			// shipped the app; recomputing them would only add churn.
			continue
		}
		if sp.integrV[0] >= 0 {
			ig, err := json.Marshal(integrity(b.data))
			if err != nil {
				return nil, err
			}
			add(sp.integrV, string(ig))
		}
	}

	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })

	out := make([]byte, len(a.rawHeader))
	copy(out, a.rawHeader)
	for _, e := range edits {
		if e.start < 0 || e.end > len(out) || e.start > e.end {
			return nil, fmt.Errorf("asar header edit %d:%d is outside the document", e.start, e.end)
		}
		tail := append([]byte{}, out[e.end:]...)
		out = out[:e.start]
		out = append(out, e.text...)
		out = append(out, tail...)
	}

	var check asarNode
	if err := json.Unmarshal(out, &check); err != nil {
		return nil, fmt.Errorf("patched asar header is not valid JSON: %w", err)
	}
	return pickleWrapJSON(out), nil
}

// pickleWrapJSON wraps a JSON document in a Chromium pickle the way
// @electron/asar does: a 4-byte payload size, then the payload, which for a
// string is a 4-byte length followed by the bytes and zero alignment padding.
// The reported payload size covers the length field and the padded string, and
// excludes the 4-byte header, which is what Pickle.toBuffer() produces.
func pickleWrapJSON(doc []byte) []byte {
	aligned := (len(doc) + 3) &^ 3
	buf := make([]byte, 8+aligned)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(4+aligned))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(doc)))
	copy(buf[8:], doc)
	return buf
}

// decodeJSONInt reads a JSON value that is either a number or a numeric string.
func decodeJSONInt(b []byte) (int64, bool) {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		return n, true
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err == nil {
			return v, true
		}
	}
	return 0, true
}

// indexNodes walks the raw header JSON and records, for every asar node, the
// byte ranges of its size, offset and integrity values.
func indexNodes(raw []byte) map[string]*span {
	p := headerParser{b: raw, spans: map[string]*span{}}
	p.i = skipWS(raw, 0)
	if p.i < len(raw) && raw[p.i] == byte('{') {
		p.parseObject("", false)
	}
	return p.spans
}

type headerParser struct {
	b     []byte
	i     int
	spans map[string]*span
}

// parseObject consumes an object. path identifies the asar node the object
// represents, or "" for the synthetic root that only holds "files".
func (p *headerParser) parseObject(path string, isNode bool) int {
	var sp *span
	if isNode {
		sp = &span{offsetV: noSpan(), sizeV: noSpan(), integrV: noSpan()}
		p.spans[path] = sp
	}
	p.i++ // consume '{'
	for {
		p.i = skipWS(p.b, p.i)
		if p.i < len(p.b) && p.b[p.i] == byte('}') {
			p.i++
			return p.i
		}
		key, next := parseString(p.b, p.i)
		if next < 0 {
			return p.i
		}
		p.i = skipWS(p.b, next)
		if p.i < len(p.b) && p.b[p.i] == byte(':') {
			p.i++
		}
		p.i = skipWS(p.b, p.i)
		start := p.i
		if key == "files" {
			// Both node objects and the synthetic root hold a files map that
			// must be descended into.
			p.parseFilesMap(path)
		} else {
			p.i = skipValue(p.b, p.i)
		}
		if isNode {
			switch key {
			case "size":
				sp.sizeV = [2]int{start, p.i}
			case "offset":
				sp.offsetV = [2]int{start, p.i}
			case "integrity":
				sp.integrV = [2]int{start, p.i}
			}
		}
		p.i = skipWS(p.b, p.i)
		if p.i < len(p.b) && p.b[p.i] == byte(',') {
			p.i++
			continue
		}
		if p.i < len(p.b) && p.b[p.i] == byte('}') {
			p.i++
			return p.i
		}
		return p.i
	}
}

// parseFilesMap consumes the object that maps child names to nodes.
func (p *headerParser) parseFilesMap(parentPath string) {
	if p.i >= len(p.b) || p.b[p.i] != byte('{') {
		p.i = skipValue(p.b, p.i)
		return
	}
	p.i++ // consume '{'
	for {
		p.i = skipWS(p.b, p.i)
		if p.i < len(p.b) && p.b[p.i] == byte('}') {
			p.i++
			return
		}
		key, next := parseString(p.b, p.i)
		if next < 0 {
			return
		}
		childPath := key
		if parentPath != "" {
			childPath = parentPath + "/" + key
		}
		p.i = skipWS(p.b, next)
		if p.i < len(p.b) && p.b[p.i] == byte(':') {
			p.i++
		}
		p.i = skipWS(p.b, p.i)
		if p.i < len(p.b) && p.b[p.i] == byte('{') {
			p.parseObject(childPath, true)
		} else {
			p.i = skipValue(p.b, p.i)
		}
		p.i = skipWS(p.b, p.i)
		if p.i < len(p.b) && p.b[p.i] == byte(',') {
			p.i++
			continue
		}
		if p.i < len(p.b) && p.b[p.i] == byte('}') {
			p.i++
			return
		}
		return
	}
}

func skipWS(b []byte, i int) int {
	for i < len(b) {
		switch b[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

func skipValue(b []byte, i int) int {
	if i >= len(b) {
		return i
	}
	switch b[i] {
	case '{', '[':
		open := b[i]
		closer := byte('}')
		if open == '[' {
			closer = ']'
		}
		depth := 0
		for i < len(b) {
			if b[i] == open {
				depth++
			} else if b[i] == closer {
				depth--
				if depth == 0 {
					return i + 1
				}
			} else if b[i] == '"' {
				_, next := parseString(b, i)
				if next < 0 {
					return len(b)
				}
				i = next
				continue
			}
			i++
		}
		return len(b)
	case '"':
		_, next := parseString(b, i)
		if next < 0 {
			return len(b)
		}
		return next
	default:
		for i < len(b) && b[i] != ',' && b[i] != '}' && b[i] != ']' {
			i++
		}
		return i
	}
}

// parseString decodes the JSON string starting at i and returns it plus the
// index just past the closing quote, so the bytes consumed line up with what
// was written. Paths with non-ASCII characters survive because bytes are copied
// verbatim apart from escapes.
func parseString(b []byte, i int) (string, int) {
	if i >= len(b) || b[i] != byte('"') {
		return "", -1
	}
	i++
	var sb strings.Builder
	for i < len(b) {
		c := b[i]
		if c == byte('"') {
			return sb.String(), i + 1
		}
		if c != byte('\\') {
			sb.WriteByte(c)
			i++
			continue
		}
		i++
		if i >= len(b) {
			break
		}
		switch b[i] {
		case 'n':
			sb.WriteByte('\n')
		case 't':
			sb.WriteByte('\t')
		case 'r':
			sb.WriteByte('\r')
		case 'b':
			sb.WriteByte('\b')
		case 'f':
			sb.WriteByte('\f')
		case '/':
			sb.WriteByte('/')
		case '\\':
			sb.WriteByte('\\')
		case '"':
			sb.WriteByte('"')
		case 'u':
			if i+4 < len(b) {
				var r rune
				ok := true
				for k := 1; k <= 4; k++ {
					d := hexDigit(b[i+k])
					if d < 0 {
						ok = false
						break
					}
					r = r<<4 | rune(d)
				}
				if ok {
					sb.WriteRune(r)
					i += 4
				}
			}
		default:
			sb.WriteByte(b[i])
		}
		i++
	}
	return sb.String(), -1
}

func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
