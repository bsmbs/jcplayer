package main

func (player *Player) PausePlay() (bool, error) {
	paused, err := player.Conn.Get("pause")
	if err != nil {
		return false, err
	}

	if paused.(bool) {
		player.Conn.Set("pause", false)
		return true, nil
		//play.SetImage(controlPause)
	} else {
		player.Conn.Set("pause", true)
		return false, nil
		//play.SetImage(controlPlay)
	}
}

func (player *Player) Seek(pos int) error {
	_, err := player.Conn.Call("seek", pos)
	if err != nil {
		return err
	}

	return nil
}
