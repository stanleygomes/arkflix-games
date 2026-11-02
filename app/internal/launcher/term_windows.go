package launcher

import "os"

func terminate(p *os.Process) error { return p.Kill() }
