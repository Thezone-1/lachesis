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
	yarn   [sceneH]string
	wheel  [sceneH]string
	thread [sceneH]string
	figure [sceneH]string
}

const (
	scW      = 7
	yarnBall = 7
	yarnW    = yarnBall + 2 // the ball plus the strand that leads to the wheel
)

// yarnFrame draws the ball of yarn. The crossing strands swap every phase and a
// bright mark travels round the rim, so it reads as rolling.
func yarnFrame(phase int) [sceneH]string {
	var rows [sceneH][]rune
	base := [2][sceneH]string{
		{` .---. `, `/ \|/ \`, `|--+--|`, `\ /|\ /`, ` '---' `},
		{` .---. `, `/ /|\ \`, `|--X--|`, `\ \|/ /`, ` '---' `},
	}[phase%2]
	for i, r := range base {
		rows[i] = []rune(r)
	}
	mark := [4][2]int{{0, 3}, {2, 6}, {4, 3}, {2, 0}}[phase%4]
	rows[mark[0]][mark[1]] = '@'
	var out [sceneH]string
	for i := range rows {
		strand := "  "
		if i == threadRow {
			strand = "~~"
		}
		out[i] = string(rows[i]) + strand
	}
	return out
}

var scissorsOpen = [3]string{
	` \ .-. `,
	`  X   )`,
	` / '-' `,
}

var scissorsShut = [3]string{
	`   .-. `,
	`==X   )`,
	`   '-' `,
}

// buildScene composes the wheel, the thread, the scissors and Atropos.
// cutFrame < 0 means no cut is happening; otherwise it is the animation frame
// (0..cutFrames-1). While waiting, the scissors sit at the end of the thread
// and snip open and shut (openIdle).
func buildScene(threadW, pos int, cat Category, idle bool, cutFrame int, ascii, openIdle bool) sceneParts {
	var p sceneParts
	p.wheel = wheelFrames[(pos/2)%2]
	p.yarn = yarnFrame(pos / 2)
	if idle {
		p.wheel = wheelFrames[0]
		p.yarn = yarnFrame(0)
	}
	p.figure = figure

	rows := make([][]rune, sceneH)
	for i := range rows {
		rows[i] = []rune(strings.Repeat(" ", threadW))
	}
	thread := []rune(threadLine(threadW, pos, cat, idle, false, ascii))
	start := threadW - scW
	mid := start / 2

	// drawThread fills the thread row over [from, to).
	drawThread := func(from, to int) {
		for i := from; i < to && i < threadW; i++ {
			if i >= 0 {
				rows[threadRow][i] = thread[i]
			}
		}
	}
	put := func(x int, sprite [3]string) {
		for r, line := range sprite {
			for c, ch := range []rune(line) {
				if x+c >= 0 && x+c < threadW && ch != ' ' {
					rows[threadRow-1+r][x+c] = ch
				} else if x+c >= 0 && x+c < threadW && r != 1 {
					rows[threadRow-1+r][x+c] = ' '
				}
			}
		}
	}

	switch {
	case cutFrame < 0 || threadW < 18:
		drawThread(0, start)
		if idle || !openIdle {
			put(start, scissorsShut)
		} else {
			put(start, scissorsOpen)
		}
	case cutFrame < 12: // walk the scissors left along the thread
		x := start - (start-mid)*cutFrame/12
		drawThread(0, threadW)
		put(x, scissorsOpen)
	case cutFrame < 20: // snip, snip
		drawThread(0, mid)
		sprite := scissorsOpen
		if (cutFrame/2)%2 == 1 {
			sprite = scissorsShut
		}
		drawThread(mid+scW, threadW)
		put(mid, sprite)
	case cutFrame < 28: // cut: the tail falls
		drawThread(0, mid)
		tailRow := threadRow + 1 + (cutFrame-20)/4
		if tailRow >= sceneH {
			tailRow = sceneH - 1
		}
		for i := mid + scW; i < threadW; i++ {
			rows[tailRow][i] = thread[i]
		}
		put(mid, scissorsShut)
	default: // scissors withdraw, the new thread follows them from the wheel
		x := mid + (start-mid)*(cutFrame-28)/(cutFrames-28)
		drawThread(0, x)
		put(x, scissorsOpen)
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
