package internal

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"git.fractalqb.de/fractalqb/xsx/gem"
	robi "github.com/go-vgo/robotgo"
)

const macroPause = 50 * time.Millisecond

func runMacro(m []gem.Expr) {
	for _, step := range m {
		log.Debuga("macro `step`", step)
		switch s := step.(type) {
		case *gem.Atom:
			if s.Quoted() {
				log.Tracea("type `string`", s.Txt)
				robi.TypeStr(s.Txt)
			} else {
				log.Tracea("tap `key`", s.Txt)
				robi.KeyTap(s.Txt)
			}
		case *gem.Sequence:
			if s.Meta() {
				log.Errora("macro has meta sequence", step)
			} else {
				switch s.Brace() {
				case gem.Square:
					playKey(s)
				case gem.Curly:
					playMouse(s)
				case gem.Paren:
					play2Proc(s)
				}
			}
		default:
			log.Errora("unhandled `element type`", reflect.TypeOf(step))
		}
		time.Sleep(macroPause) // TODO make it adjustable
	}
}

func playKey(m *gem.Sequence) {
	if len(m.Elems) == 0 {
		log.Warns("empty key sequence in")
		return
	}
	var cmd []string
	action := 0
	modsAt := 1
	e := m.Elems[0].(*gem.Atom)
	if e.Meta() {
		switch e.Txt {
		case "down":
			action = -1
			cmd = append(cmd, "down")
		case "up":
			action = 1
			cmd = append(cmd, "up")
		case "tap":
			action = 0
		default:
			log.Errora("unknown `key action`", e.Txt)
			return
		}
		if len(m.Elems) < 2 {
			log.Errora("missing key spec in `key sequence`", m)
		}
		cmd = append(cmd, m.Elems[1].(*gem.Atom).Txt)
		modsAt = 2
	} else {
		cmd = append(cmd, e.Txt)
	}
	for _, e := range m.Elems[modsAt:] {
		cmd = append(cmd, e.(*gem.Atom).Txt)
	}
	switch action {
	case 0:
		mods := make([]interface{}, len(cmd))
		for i := range cmd {
			mods[i] = cmd[i]
		}
		log.Tracea("tap `key` with `mods`", cmd[0], mods[1:])
		robi.KeyTap(cmd[0], mods[1:]...)
	default:
		cmd[0], cmd[1] = cmd[1], cmd[0]
		log.Tracea("toggle `key` with `mods`", cmd[0], cmd[1:])
		robi.KeyToggle(cmd[0], cmd[1:]...)
	}
}

func playMouse(m *gem.Sequence) {
	for ip := 0; ip < len(m.Elems); ip++ {
		switch m.Elems[ip].(*gem.Atom).Txt {
		case "left":
			ip++
			mouseButton("left", m.Elems[ip].(*gem.Atom).Txt)
		case "middle":
			ip++
			mouseButton("center", m.Elems[ip].(*gem.Atom).Txt)
		case "right":
			ip++
			mouseButton("right", m.Elems[ip].(*gem.Atom).Txt)
		case "click":
			ip++
			xk, yk := mouseCoos(
				m.Elems[ip].(*gem.Atom).Txt,
				m.Elems[ip+1].(*gem.Atom).Txt)
			ip++
			robi.MoveMouse(xk, yk)
		case "drag":
			ip++
			xk, yk := mouseCoos(
				m.Elems[ip].(*gem.Atom).Txt,
				m.Elems[ip+1].(*gem.Atom).Txt)
			ip++
			robi.DragMouse(xk, yk)
		case "scroll":
			ip++
			count, _ := strconv.ParseInt(m.Elems[ip].(*gem.Atom).Txt, 10, 32)
			ip++
			dir := m.Elems[ip].(*gem.Atom).Txt
			robi.ScrollMouse(int(count), dir)
		default:
			log.Errora("unknown `mouse action`", m.Elems[ip].(*gem.Atom).Txt)
		}
	}
}

func mouseCoos(xStr, yStr string) (x int, y int) {
	xpf := strings.ContainsAny(xStr, "+-")
	ypf := strings.ContainsAny(yStr, "+-")
	if xpf || ypf {
		x, y = robi.GetMousePos()
		if xpf {
			tmp, _ := strconv.ParseInt(xStr[1:], 10, 32)
			if xStr[0] == '+' {
				x += int(tmp)
			} else {
				x -= int(tmp)
			}
		} else {
			tmp, _ := strconv.ParseInt(xStr, 10, 32)
			x = int(tmp)
		}
		if ypf {
			tmp, _ := strconv.ParseInt(yStr[1:], 10, 32)
			if yStr[0] == '+' {
				y += int(tmp)
			} else {
				y -= int(tmp)
			}
		} else {
			tmp, _ := strconv.ParseInt(yStr, 10, 32)
			y = int(tmp)
		}
	} else {
		px, err := strconv.ParseInt(xStr, 10, 32)
		if err != nil {
			log.Errora("parse mouse x-coo '%s'", xStr)
		}
		py, err := strconv.ParseInt(yStr, 10, 32)
		if err != nil {
			log.Errora("parse mouse y-coo '%s'", yStr)
		}
		x = int(px)
		y = int(py)
	}
	return x, y
}

func mouseButton(which string, action string) {
	switch action {
	case "click":
		robi.MouseClick(which, false)
	case "double":
		robi.MouseClick(which, true)
	case "down":
		robi.MouseToggle("down", which)
	case "up":
		robi.MouseToggle("up", which)
	default:
		log.Errora("unknown `mouse-button action`", action)
	}
}

func play2Proc(s *gem.Sequence) {
	if len(s.Elems) > 0 {
		// TODO: switching seems to not yet work?
		procNm := s.Elems[0].(*gem.Atom).Txt
		log.Debuga("macro switch to `process`", procNm)
		current := robi.GetActive()
		robi.ActiveName(procNm)
		defer func() {
			log.Debuga("macro switch back from `process`", procNm)
			robi.SetActive(current)
		}()
		runMacro(s.Elems[1:])
	}
}
