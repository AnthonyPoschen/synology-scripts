package main

import (
		//"time"
		//"fmt"
	"github.com/google/gxui"
	"github.com/google/gxui/drivers/gl"
	"github.com/google/gxui/gxfont"
	//"github.com/google/gxui/math"
	"github.com/google/gxui/themes/dark"
)

func appMain(driver gxui.Driver) {
	theme := dark.CreateTheme(driver)

	font, err := driver.CreateFont(gxfont.Default, 16)
	if err != nil {
		panic(err)
	}

	window := theme.CreateWindow(900, 600, "hello World")
	window.SetBackgroundBrush(gxui.CreateBrush(gxui.Gray50))
	
	ce := theme.CreateCodeEditor()
	ce.SetDesiredWidth(900)
	ce.SetFont(font)
	window.AddChild(ce)
	window.OnClose(driver.Terminate)
}

func main() {
	gl.StartDriver(appMain)
}

