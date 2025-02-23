package main

import (
	"github.com/SongZihuan/ssh-watcher/src/mainfunc/sshwatcher"
	"github.com/SongZihuan/ssh-watcher/src/utils"
)

func main() {
	utils.Exit(sshwatcher.MainV1())
}
