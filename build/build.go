package build

type Build struct {
	
}

type BuildManager struct {
  builds []*Build
}

func NewBuildManager() {
	return &BuildManager{}
}

func NewBuild(title string) (*Build, error) {
	if title == "" {
		return nil, fmt.Errorf("empty title")
	}
	return &Build{0, title, false}, nil
}

func (m *BuildManager) All() []*Task {
	return m.builds
}