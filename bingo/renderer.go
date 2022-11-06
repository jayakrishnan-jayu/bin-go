package bingo

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"text/tabwriter"
)

func ClearTerminal() {
	switch runtime.GOOS {
	case "darwin":
		runCmd("clear")
	case "linux":
		runCmd("clear")
	case "windows":
		runCmd("cmd", "/c", "cls")
	default:
		runCmd("clear")
	}
}

func runCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func (p PlayersList) RenderLobby() {
	ClearTerminal()
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 1)
	fmt.Println("Lobby")
	for _, p := range p.Players {
		fmt.Fprintf(w, "%d)\t%s\t(%s)\n", p.Id, p.Name, p.Ip)
	}
	w.Flush()
}
