package dto

type Model struct {
	RegisterLength uint16
	Properties     []*Property
}

type Property struct {
	Type string
}

type Device struct {
	Model *Model
}

type PropertyMap struct {
	m map[uint32]PropertyUnit
}

type RegisterMap struct {
}

type PropertyUnit struct {
	Index  uint16
	Length uint16
	Value  []uint16
}

func (p *PropertyMap) Get(did string, pid string) (PropertyUnit, error) {

}

type RegisterManager struct {
	models  map[string]*Model
	devices map[string]*Device
}
