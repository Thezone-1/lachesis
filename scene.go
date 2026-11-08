package main

import "strings"

const (
	wheelW     = 11
	figureW    = 7
	sceneH     = 5
	cutFrames  = 44 // length of the scissors animation, in frames
	threadRow  = 2
	cutFromEnd = 6 // how far from the figure the scissors start
)

var wheelFrames = [2][sceneH]string{
	{
		`  .-"""-.  `,
		` /   |   \ `,
		`|  --+--  |`,
		` \   |   / `,
		`  '-...-'  `,
	},
	{
		`  .-"""-.  `,
		` / \   / \ `,
		`|   \ /   |`,
		` \  / \  / `,
		`  '-...-'  `,
	},
}

var figure = [sceneH]string{
	` .---. `,
	`( o o )`,
	` \_-_/ `,
	`  /|\  `,
	`  / \  `,
}

// sceneParts are the drawn rows of the wheel, the thread area and the figure.
// They are kept apart so each can be coloured on its own.
type sceneParts struct {
	wheel  [sceneH]string
	thread [sceneH]string
	figure [sceneH]string
}

// buildScene composes the wheel, the thread and Atropos. cutFrame < 0 means no
// cut is happening; otherwise it is the animation frame (0..cutFrames-1).
func buildScene(threadW, pos int, cat Category, idle bool, cutFrame int, ascii bool) sceneParts {
	var p sceneParts
	p.wheel = wheelFrames[(pos/2)%2]
	if idle {
		p.wheel = wheelFrames[0]
	}
	p.figure = figure

	rows := make([][]rune, sceneH)
	for i := range rows {
		rows[i] = []rune(strings.Repeat(" ", threadW))
	}
	copy(rows[threadRow], []rune(threadLine(threadW, pos, cat, idle, false, ascii)))

	if cutFrame >= 0 && threadW > 16 {
		mid := threadW / 2
		start := threadW - cutFromEnd
		open, closed := "✂", "✂"
		if ascii {
			open, closed = "8<", "8-"
		}
		x, sym := start, open
		switch {
		case cutFrame < 12: // walk the scissors left along the thread
			x = start - (start-mid)*cutFrame/12
		case cutFrame < 20: // snip, snip
			x = mid
			if (cutFrame/2)%2 == 1 {
				sym = closed
			}
		case cutFrame < 28: // thread is cut; the tail falls
			x, sym = mid, closed
			for i := mid + 1; i < threadW; i++ {
				rows[threadRow][i] = ' '
			}
			fall := (cutFrame - 20) / 3
			if fall > 2 {
				fall = 2
			}
			tail := []rune(strings.Repeat(" ", threadW))
			for i := mid + 1; i < threadW && i < mid+10; i++ {
				tail[i] = '~'
			}
			if threadRow+1+fall < sceneH {
				copy(rows[threadRow+1+fall][mid+1:], tail[mid+1:])
			}
		default: // scissors withdraw; the thread is re-spun from the wheel
			back := (cutFrame - 28) * (start - mid) / 16
			x, sym = mid+back, open
			if x > start {
				x = start
			}
			for i := mid + 1; i < threadW; i++ {
				if i < x {
					rows[threadRow][i] = ' '
				}
			}
		}
		for i, r := range []rune(sym) {
			if x+i >= 0 && x+i < threadW {
				rows[threadRow][x+i] = r
			}
		}
	}
	for i := range rows {
		p.thread[i] = string(rows[i])
	}
	return p
}

// cutLabel is the line shown while Atropos works.
func cutLabel(cutFrame int, app string) string {
	switch {
	case cutFrame < 0:
		return ""
	case cutFrame < 12:
		return "Atropos walks the thread to cut " + app + "..."
	case cutFrame < 20:
		return "snip, snip..."
	case cutFrame < 28:
		return "the " + app + " thread is cut"
	default:
		return "Lachesis spins a new thread"
	}
}
