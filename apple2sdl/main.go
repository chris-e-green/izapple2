package main

import (
	"unsafe"

	"github.com/ivanizag/apple2"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	a := apple2.MainApple()
	SDLRun(a)
}

// SDLRun starts the Apple2 emulator on SDL
func SDLRun(a *apple2.Apple2) {
	s := newSDLSpeaker()
	s.start()

	window, renderer, err := sdl.CreateWindowAndRenderer(4*40*7, 4*24*8,
		sdl.WINDOW_SHOWN)
	if err != nil {
		panic("Failed to create window")
	}
	window.SetResizable(true)

	defer window.Destroy()
	defer renderer.Destroy()
	window.SetTitle("Apple2")

	kp := newSDLKeyBoard(a)
	a.SetKeyboardProvider(kp)
	a.SetSpeakerProvider(s)
	go a.Run(false)

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				//fmt.Printf("[%d ms] Keyboard\ttype:%d\tsym:%c\tmodifiers:%d\tstate:%d\trepeat:%d\n",
				//	t.Timestamp, t.Type, t.Keysym.Sym, t.Keysym.Mod, t.State, t.Repeat)
				kp.putKey(t)
			case *sdl.TextInputEvent:
				//fmt.Printf("[%d ms] TextInput\ttype:%d\texts:%s\n",
				//	t.Timestamp, t.Type, t.GetText())
				kp.putText(t)
			}
		}

		img := apple2.Snapshot(a)
		if img != nil {
			surface, err := sdl.CreateRGBSurfaceFrom(unsafe.Pointer(&img.Pix[0]),
				int32(img.Bounds().Dx()), int32(img.Bounds().Dy()),
				32, 4*img.Bounds().Dx(),
				0x0000ff, 0x0000ff00, 0x00ff0000, 0xff000000)
			// Valid for little endian. Should we reverse for big endian?
			// 0xff000000, 0x00ff0000, 0x0000ff00, 0x000000ff)

			if err != nil {
				panic(err)
			}

			texture, err := renderer.CreateTextureFromSurface(surface)
			if err != nil {
				panic(err)
			}

			renderer.Clear()
			w, h := window.GetSize()
			renderer.Copy(texture, nil, &sdl.Rect{X: 0, Y: 0, W: w, H: h})
			renderer.Present()

			surface.Free()
			texture.Destroy()
		}
		sdl.Delay(1000 / 30)
	}

}