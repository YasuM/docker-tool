package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"io"

	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

var (
	rootFlex      *tview.Flex        = tview.NewFlex()
	rightFlex     *tview.Flex        = tview.NewFlex()
	app           *tview.Application = tview.NewApplication()
	leftMenuList  *tview.List        = tview.NewList()
	rightList     *tview.List        = tview.NewList()
	modal         *tview.Modal       = tview.NewModal()
	errorTextview *tview.TextView    = tview.NewTextView()
)

func main() {
	leftMenuList.AddItem("Images", "", '1', nil).
		AddItem("Containers", "", '2', nil).
		AddItem("Volumes", "", '3', nil).
		SetChangedFunc(func(i int, mainText string, secondaryText string, shortcut rune) {
			if i == 0 {
				handlerImage()
			} else if i == 1 {
				handlerContainer()
			} else if i == 2 {
				handlerVolume()
			}
		}).
		SetBorder(true)

	rightList.SetBorder(true)
	errorTextview.SetBorder(true)

	rightFlex.SetDirection(tview.FlexRow).
		AddItem(rightList, 0, 10, false).
		AddItem(errorTextview, 0, 1, false)

	setInputCaptureOn()

	rootFlex.AddItem(leftMenuList, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)
	if err := app.SetRoot(rootFlex, true).SetFocus(leftMenuList).Run(); err != nil {
		panic(err)
	}
}

func setInputCaptureOn() {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRight {
			app.SetFocus(rightList)
		} else if event.Key() == tcell.KeyLeft {
			app.SetFocus(leftMenuList)
		}
		return event
	})

}

func setInputCaptureOff() {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return event
	})
}

func handlerImage() {
	rootFlex.RemoveItem(modal)
	// app.SetFocus(rightList)
	rightList.Clear()
	initModalInit()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	ctx := context.Background()
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		iID := image.ID
		item := fmt.Sprintf("%s", image.RepoTags[0])
		rightList.AddItem(item, "", 'i', func() {
			rootFlex.RemoveItem(rightFlex)
			modal.SetText(iID).
				AddButtons([]string{"run", "remove"}).
				SetDoneFunc(func(_ int, buttonLabel string) {
					if buttonLabel == "remove" {
						_, err = cli.ImageRemove(ctx, iID, types.ImageRemoveOptions{})
						if err != nil {
							errorTextview.SetText(err.Error())
						}
					} else if buttonLabel == "run" {
						resp, err := cli.ContainerCreate(ctx, &container.Config{Image: image.RepoTags[0]}, nil, nil, nil, "")
						if err != nil {
							errorTextview.SetText(err.Error())
						}
						cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
					} else {
						errorTextview.SetText("")
					}
					setInputCaptureOn()
					rootFlex.AddItem(rightFlex, 0, 4, false)
					handlerImage()
				})
			rootFlex.AddItem(modal, 0, 4, false)
			app.SetFocus(modal)
			setInputCaptureOff()
		})
	}
}

func handlerContainer() {
	rootFlex.RemoveItem(modal)
	// app.SetFocus(rightList)
	rightList.Clear()
	initModalInit()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}

	for _, c := range containers {
		str := fmt.Sprintf("%s(%s) %s", c.Image, c.ID, c.Status)
		cID := c.ID
		rightList.AddItem(str, "", 'c', func() {
			rootFlex.RemoveItem(rightFlex)
			modal.SetText(cID + " Delete?").
				AddButtons([]string{"remove", "stop", "logs"}).
				SetDoneFunc(func(_ int, buttonLabel string) {
					ctx := context.Background()
					if buttonLabel == "remove" {
						err := cli.ContainerRemove(ctx, cID, types.ContainerRemoveOptions{})
						if err != nil {
							errorTextview.SetText(err.Error())
						}
					} else if buttonLabel == "stop" {
						err := cli.ContainerStop(ctx, cID, container.StopOptions{})
						if err != nil {
							errorTextview.SetText(err.Error())
						}
					} else if buttonLabel == "logs" {
						reader, err := cli.ContainerLogs(ctx, cID, types.ContainerLogsOptions{ShowStdout: true, Details: true, ShowStderr: true})
						if err != nil {
							errorTextview.SetText(err.Error())
						}
						a, _ := io.ReadAll(reader)
						errorTextview.SetText(string(a))
					}

					rootFlex.AddItem(rightFlex, 0, 4, false)
					setInputCaptureOn()
					handlerContainer()
				})
			rootFlex.AddItem(modal, 0, 4, false)
			app.SetFocus(modal)
			setInputCaptureOff()
		})
	}
}

func handlerVolume() {
	rightList.Clear()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	volumes, _ := cli.VolumeList(context.Background(), volume.ListOptions{})
	for _, v := range volumes.Volumes {
		name := v.Name
		rightList.AddItem(name, "", 'v', nil)
	}
}

func initModalInit() {
	modal.ClearButtons()
	modal.SetText("")
}
