package brazeclient

func NewOptNilPointerString(v *string) OptNilString {
	ons := OptNilString{}
	if v != nil {
		ons.SetTo(*v)
	} else {
		ons.SetToNull()
	}

	return ons
}

func (o OptNilString) GetPointer() *string {
	if !o.Set || o.Null {
		return nil
	}

	return &o.Value
}
