package gofpdi

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

// mixedPDF builds a minimal PDF whose pages have different MediaBoxes; page 3
// also carries /Rotate -90.
func mixedPDF() []byte {
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 4 0 R 6 0 R] /Count 3 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 200] /Resources << >> /Contents 5 0 R >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 150] /Resources << >> /Contents 5 0 R >>",
		"<< /Length 0 >>\nstream\n\nendstream",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 150] /Rotate -90 /Resources << >> /Contents 5 0 R >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func newMixedImporter() *Importer {
	rs := io.ReadSeeker(bytes.NewReader(mixedPDF()))
	imp := NewImporter()
	imp.SetSourceStream(&rs)
	return imp
}

func TestImportPageUsesItsOwnBox(t *testing.T) {
	imp := newMixedImporter()
	tpl1 := imp.ImportPage(1, "/MediaBox")
	tpl2 := imp.ImportPage(2, "/MediaBox")
	w := imp.GetWriter()

	for tpl, want := range map[int][2]float64{tpl1: {100, 200}, tpl2: {300, 150}} {
		box := w.tpls[tpl].Box
		if box["w"] != want[0] || box["h"] != want[1] {
			t.Errorf("template %d box is %vx%v, want %vx%v", tpl, box["w"], box["h"], want[0], want[1])
		}
	}

	// Placing page 2 at a given width must keep page 2's aspect ratio,
	// not page 1's: 600 wide -> 300 high.
	_, scaleX, scaleY, _, _ := imp.UseTemplate(tpl2, 0, 0, 600, 0)
	if scaleX != 2 || scaleY != 2 {
		t.Errorf("page 2 at width 600: scale %vx%v, want 2x2", scaleX, scaleY)
	}
}

func TestGetPageRotations(t *testing.T) {
	got := newMixedImporter().GetPageRotations()
	if got[1] != 0 || got[2] != 0 || got[3] != 270 {
		t.Errorf("rotations = %v, want map[1:0 2:0 3:270]", got)
	}
}
