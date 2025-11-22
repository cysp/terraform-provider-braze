package brazeclient

func NewOptPointerString(v *string) OptString {
	o := OptString{}
	if v != nil {
		o.SetTo(*v)
	}

	return o
}

func (o OptString) GetPointer() *string {
	if !o.Set {
		return nil
	}

	return &o.Value
}
