#!/usr/bin/env wish

package require Tk

# Global variables
set ::prefix "$::env(HOME)/.local"
set ::pipe {}
set ::running 0
set ::current_target {}

# Dark theme colors
set dark_bg #1e1e1e
set mid_bg #2d2d30
set light_fg #dcdcdc
set white_fg #ffffff
set entry_bg #3c3f41
set entry_sel_bg #007acc
set entry_sel_fg #ffffff
set btn_bg #404144
set btn_active_bg #525558
set status_bg #252526
set status_fg #cccccc
set prog_bg #0f7dc6
set prog_light #4da6ff
set prog_dark #095aa0

ttk::style theme use clam
ttk::style configure TProgressbar -background $prog_bg -lightcolor $prog_light -darkcolor $prog_dark -troughcolor $dark_bg -bordercolor $mid_bg

# Create widgets first
text .output -yscrollcommand {.scroll set} -bg $dark_bg -fg $light_fg -relief sunken -bd 2 -font {-family monospace -size 10} -insertbackground $white_fg
.output yview moveto 1.0
scrollbar .scroll -orient vertical -command {.output yview} -bg $dark_bg -activebackground $btn_active_bg -troughcolor $mid_bg
entry .prefix_entry -textvariable ::prefix -width 50 -bg $entry_bg -fg $light_fg -selectbackground $entry_sel_bg -selectforeground $entry_sel_fg -insertbackground $white_fg
label .prefix_label -text "INSTALL_PREFIX (for install):" -bg $mid_bg -fg $white_fg
label .build_label -text "Build & Install Targets:" -bg $mid_bg -fg $white_fg
frame .build_buttons -bg $mid_bg
label .info_label -text "Info Targets:" -bg $mid_bg -fg $white_fg
frame .info_buttons -bg $mid_bg
label .quality_label -text "Code Quality Targets:" -bg $mid_bg -fg $white_fg
frame .quality_buttons -bg $mid_bg
ttk::progressbar .progress -mode indeterminate -length 400
label .status -text "Ready" -relief sunken -anchor w -bg $status_bg -fg $status_fg
button .clear -text "Clear Output" -command { .output delete 1.0 end } -pady 10 -relief flat -bd 1 -bg $btn_bg -fg $white_fg -activebackground $btn_active_bg -activeforeground $white_fg

# Load and scale logo to max 64x64 preserving aspect ratio
set src [image create photo -file "gopherutils.png"]
set ow [image width $src]
set oh [image height $src]
set factorx [expr ($ow + 63) / 64]
set factory [expr ($oh + 63) / 64]
set subsample [expr {$factorx > $factory ? $factorx : $factory}]
image create photo ::logo
::logo copy $src -subsample $subsample $subsample
image delete $src
label .logo_label -image ::logo -anchor center -bg $mid_bg

# Procedure to run make command (blocking, captures stdout+stderr)
proc start_make {target} {
    if { [ info exists ::pipe ] } {
        if { $::pipe ne {} } {
            bell
            return
        }
    }
    set ::current_target $target
    set ::running 1
    .output insert end "\n=== Running: make $target ===\n"
    .output yview moveto 1.0
    update idletasks
    
    set long_tasks {test coverage CQ build deps install uninstall lint staticcheck}
    if { [lsearch -exact $long_tasks $target] >= 0 } {
        .progress start
    }
    
    set cmd [list make $target]
    if {$target eq "install"} {
        lappend cmd "INSTALL_PREFIX=$::prefix"
    }
    set pipe_cmd "| [join $cmd { }] 2>&1"
    set ::pipe [open $pipe_cmd r]
    fconfigure $::pipe -buffering none -blocking 0
    fileevent $::pipe readable read_pipe
}

proc read_pipe {} {
    set fh $::pipe
    set data [read $fh 4096]
    if {$data ne ""} {
        .output insert end $data
        .output yview moveto 1.0
        update idletasks
    }
    if {[eof $fh]} {
        set status [catch {close $fh} result]
        unset ::pipe
        set ::running 0
        .progress stop
        if {$status} {
            .output insert end "\nError (code $::errorCode): $result\n--- Non-zero exit ---\n"
        } else {
            .output insert end "\n--- Exit code: 0 ---\n\n\n"
        }
        .output yview moveto 1.0
        .status configure -text "Done: make $::current_target"
        unset ::current_target
        return
    }
}

# Build Tools buttons
foreach tgt {build deps clean install uninstall} {
    set btn_name [string tolower $tgt]
    button .build_buttons.$btn_name -text [string toupper $tgt] -command [list start_make $tgt] -pady 1 -bg $btn_bg -fg $white_fg -activebackground $btn_active_bg -activeforeground $white_fg -relief flat -bd 1
    pack .build_buttons.$btn_name -side left -padx 3
}

# Info Tools buttons
foreach tgt {list help test coverage} {
    set btn_name [string tolower $tgt]
    button .info_buttons.$btn_name -text [string toupper $tgt] -command [list start_make $tgt] -pady 1 -bg $btn_bg -fg $white_fg -activebackground $btn_active_bg -activeforeground $white_fg -relief flat -bd 1
    pack .info_buttons.$btn_name -side left -padx 3
}

# Code quality buttons
foreach tgt {fmt fmt-check lint vet staticcheck CQ} {
    set btn_name [string map {- _} [string tolower $tgt]]
    button .quality_buttons.$btn_name -text [string toupper $tgt] -command [list start_make $tgt] -pady 1 -bg $btn_bg -fg $white_fg -activebackground $btn_active_bg -activeforeground $white_fg -relief flat -bd 1
    pack .quality_buttons.$btn_name -side left -padx 3
}

# Layout (top-to-bottom, status bottom)
pack .logo_label -side top -pady 5
pack .prefix_label -side top -anchor w -pady 2
pack .prefix_entry -side top -fill x -pady 2
pack .build_label -side top -anchor w -pady 1
pack .build_buttons -side top -fill x -pady 2
pack .info_label -side top -anchor w -pady 1
pack .info_buttons -side top -fill x -pady 2
pack .quality_label -side top -anchor w -pady 1
pack .quality_buttons -side top -fill x -pady 2
pack .progress -side top -fill x -pady 1
frame .output_frame -bg $dark_bg
lower .output_frame
pack .clear -in .output_frame -side bottom -fill x -pady 2
pack .output -in .output_frame -side left -fill both -expand 1
pack .scroll -in .output_frame -side right -fill y
pack .output_frame -side top -fill both -expand 1 -pady 2
pack .status -side bottom -fill x

.status configure -text "gopherutils Makefile GUI - Run from project root"
wm title . "gopherutils Makefile GUI"
wm resizable . 1 1
wm geometry . 1400x1200
wm iconphoto . -default ::logo
. configure -bg $dark_bg
.output yview moveto 1.0

bind . <Escape> { exit }

# console show  ;# requires Tkcon package
