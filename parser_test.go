package sugar

import (
	"testing"
)

type GetUserLevel struct {
	Get   string `match:"k"`
	User  int
	Level int `match:"k"`
}

type GetGamerRmb struct {
	Get  string `match:"k"`
	User int
	Rmb  int `match:"k"`
}

func Test_CmdParser(t *testing.T) {
	pm := Parser{Type: ParserTypeCmd}
	pm.RegisterMsg(&GetUserLevel{}, nil)

	p := pm.Get()
	m, _ := p.ParseC2S(NewStrMsg("get user 1 level"))
	t.Logf("%#v\n", m.C2S().(*GetUserLevel))

	t.Log(m.C2SString())
}
