package main

// On macOS set terminal in ~/.config/mountly/config.json.
// Example: "terminal": "open -a iTerm"
// Note: the ssh command is appended as arguments, so the terminal must support
// receiving a command (e.g. iTerm: "open -a iTerm --args -e").
const defaultTerminal = ""
