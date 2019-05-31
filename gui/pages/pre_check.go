// Copyright © 2019 Intel Corporation
//
// SPDX-License-Identifier: GPL-3.0-only

package pages

import (
	"strings"
	"time"

	"github.com/gotk3/gotk3/gtk"

	ctrl "github.com/clearlinux/clr-installer/controller"
	"github.com/clearlinux/clr-installer/gui/common"
	"github.com/clearlinux/clr-installer/log"
	"github.com/clearlinux/clr-installer/model"
	"github.com/clearlinux/clr-installer/progress"
	"github.com/clearlinux/clr-installer/utils"
)

// PreCheckPage is a specialised page type with no corresponding ContentView summary. It handles the pre-check routine.
type PreCheckPage struct {
	controller Controller
	model      *model.SystemInstall
	layout     *gtk.Box

	pbar      *gtk.ProgressBar    // Progress bar
	list      *gtk.ListBox        // Scrolling list for messages
	selection int                 // Current progress selection
	scroll    *gtk.ScrolledWindow // Hold the list

	widgets map[int]*InstallWidget // Mapping of widgets
	info    *gtk.Label             // Display info during pre-check
}

// NewPreCheckPage constructs a new PreCheckPage.
func NewPreCheckPage(controller Controller, model *model.SystemInstall) (Page, error) {
	var err error

	// Create page
	page := &PreCheckPage{
		controller: controller,
		model:      model,
		widgets:    make(map[int]*InstallWidget),
		selection:  -1,
	}

	// Create layout
	page.layout, err = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		return nil, err
	}

	// Create scroller
	page.scroll, err = gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		return nil, err
	}
	page.scroll.SetMarginStart(48)
	page.scroll.SetMarginEnd(48)
	page.scroll.SetMarginTop(24)
	page.scroll.SetMarginBottom(24)
	page.layout.PackStart(page.scroll, true, true, 0)

	// Create list
	page.list, err = gtk.ListBoxNew()
	if err != nil {
		return nil, err
	}
	page.list.SetSelectionMode(gtk.SELECTION_NONE)
	page.scroll.Add(page.list)
	st, err := page.list.GetStyleContext()
	if err != nil {
		return nil, err
	}
	st.AddClass("scroller-main")

	page.info, err = setLabel("", "label-info", 0)
	if err != nil {
		return nil, err
	}
	page.info.SetMarginStart(24)
	page.info.SetMarginEnd(24)
	page.info.SetMaxWidthChars(1) // The value does not matter but its required for LineWrap to work
	page.info.SetLineWrap(true)
	page.layout.PackStart(page.info, false, false, 0)

	// Create progress bar
	page.pbar, err = gtk.ProgressBarNew()
	if err != nil {
		return nil, err
	}
	page.pbar.SetHAlign(gtk.ALIGN_FILL)
	page.pbar.SetMarginStart(24)
	page.pbar.SetMarginEnd(24)
	page.pbar.SetMarginBottom(12)
	page.pbar.SetMarginTop(12)
	page.layout.PackEnd(page.pbar, false, false, 0)

	return page, nil
}

// IsRequired is just here for the Page API
func (page *PreCheckPage) IsRequired() bool {
	return true
}

// IsDone is just here for the Page API
func (page *PreCheckPage) IsDone() bool {
	return false
}

// GetID returns the ID for this page
func (page *PreCheckPage) GetID() int {
	return PageIDPreCheck
}

// GetSummary returns the summary for this page
func (page *PreCheckPage) GetSummary() string {
	return page.GetTitle()
}

// GetTitle returns the title for this page
func (page *PreCheckPage) GetTitle() string {
	return utils.Locale.Get("Checking Prerequisites")
}

// GetIcon returns the icon for this page
func (page *PreCheckPage) GetIcon() string {
	return "emblem-system-symbolic"
}

// GetConfiguredValue returns nothing here
func (page *PreCheckPage) GetConfiguredValue() string {
	return ""
}

// GetRootWidget returns the root embeddable widget for this page
func (page *PreCheckPage) GetRootWidget() gtk.IWidget {
	return page.layout
}

// StoreChanges is just here for the Page API
func (page *PreCheckPage) StoreChanges() {}

// ResetChanges begins the pre-check
func (page *PreCheckPage) ResetChanges() {
	msg := utils.Locale.Get(utils.Locale.Get("Checking Prerequisites."))
	msg = msg + " " + utils.Locale.Get("Please wait.")
	page.info.SetText(msg)

	go func() {
		var success bool
		progress.Set(page)
		err := ctrl.PreCheck(page.model)
		if err != nil {
			text := utils.Locale.Get("Prerequisites for installation are not met.")
			text = text + "\n" + strings.Split(err.Error(), "\n")[0]
			page.info.SetText(text)
			sc, err := page.info.GetStyleContext()
			if err != nil {
				log.Warning("Error getting style context: ", err) // Just log trivial error
			} else {
				sc.RemoveClass("label-info")
				sc.AddClass("label-warning")
			}
			success = false
		} else {
			text := utils.Locale.Get("Prerequisites for installation are met. Proceeding.")
			page.info.SetText(text)
			// On success, disable exit button so that the user does not click it accidentally
			page.controller.SetButtonState(ButtonExit, false)
			success = true
		}
		page.pbar.SetFraction(1.0)
		time.Sleep(common.LoopWaitDuration * 25) // Wait for a while so that the user can read the message
		page.controller.SetPreCheckChannel(success)
	}()
}

// Desc pushes a description box into the view for later marking
func (page *PreCheckPage) Desc(desc string) {
	// Increment selection
	page.selection++

	// do we have an old widget? if so, mark complete
	if page.selection > 0 {
		page.widgets[page.selection-1].Completed()
	}

	// Create new install widget
	widg, err := NewInstallWidget(desc)
	if err != nil {
		panic(err)
	}
	page.widgets[page.selection] = widg

	// Pack it into the list
	page.list.Add(widg.GetRootWidget())

	// Scroll to the new item
	row := page.list.GetRowAtIndex(page.selection)
	page.list.SelectRow(row)
	scrollToView(page.scroll, page.list, &row.Widget)
}

// Failure handles failure to install
func (page *PreCheckPage) Failure() {
	page.widgets[page.selection].MarkStatus(false)
}

// Success notes the install was successful
func (page *PreCheckPage) Success() {
	page.widgets[page.selection].MarkStatus(true)
}

// LoopWaitDuration will return the duration for step-waits
func (page *PreCheckPage) LoopWaitDuration() time.Duration {
	return common.LoopWaitDuration
}

// Partial handles an actual progress update
func (page *PreCheckPage) Partial(total int, step int) {
	page.pbar.SetFraction(float64(step) / float64(total))
}

// Step will step the progressbar in indeterminate mode
func (page *PreCheckPage) Step() {
	// Pulse twice for visual feedback
	page.pbar.Pulse()
	page.pbar.Pulse()
}
