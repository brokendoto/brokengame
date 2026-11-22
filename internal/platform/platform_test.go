//go:build !steam

package platform

import "testing"

// Без тега steam игра работает на заглушке, и её методы ничего не ломают.
func TestNone(t *testing.T) {
	if Current.Name() != "" || Current.PlayerName() != "" {
		t.Fatal("заглушка должна быть безымянной")
	}
	Current.Unlock("first_win")
	Current.SetStatus("test")
	Current.Update()
}
