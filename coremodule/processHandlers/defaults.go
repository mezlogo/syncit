package processhandlers

var defaultHandlers = map[string]string{
	"pacman":      "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE",
	"aur":         "yay -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE",
	"systemd":     "sudo systemctl enable $UNIQUE_SINGLE_VALUE",
	"usersystemd": "systemctl --user enable $UNIQUE_SINGLE_VALUE",
}
