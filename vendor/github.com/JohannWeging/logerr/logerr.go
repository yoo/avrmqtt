package logerr

// Error holds the added fields
type Error struct {
	prev   error
	Fields map[string]interface{}
}

// Fields are the log fields
type Fields map[string]interface{}

// WithField adds a single field to the error
func WithField(err error, key string, value interface{}) error {
	if err == nil {
		return nil
	}
	return &Error{
		prev:   err,
		Fields: Fields{key: value},
	}
}

// WithFields adds a new field map to the error
func WithFields(err error, fields map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return &Error{
		prev:   err,
		Fields: fields,
	}
}

func DeferWithFields(err *error, fields map[string]interface{}) {
	if *err == nil {
		return
	}
	*err = WithFields(*err, fields)
}

// GetFields returns the log fields from the wrapped errors
func GetFields(err error) map[string]interface{} {
	fields := map[string]interface{}{}
	errs := unwrap(err)
	for _, e := range errs {
		logErr, ok := e.(*Error)
		if !ok {
			continue
		}
		for k, v := range logErr.Fields {
			fields[k] = v
		}
	}
	return fields
}

// Error returns the error message of the wrapped errors
func (e *Error) Error() string {
	if e.prev == nil {
		return ""
	}
	return e.prev.Error()
}

// Underlying implements the juju/errors wrapper interface
func (e *Error) Underlying() error {
	return e.prev
}

// Cause implements the pkg/errors causer interface
func (e *Error) Cause() error {
	return e.prev
}

func unwrap(err error) []error {
	errs := []error{err}
	// the causer interface is checked last because of its ambiguous behaviour
	// caused by the different implementations in juju/errors and pkg/errors
	switch e := err.(type) {
	case wrapper:
		errs = append(errs, unwrap(e.Underlying())...)
	case causer:
		errs = append(errs, unwrap(e.Cause())...)
	}
	return errs
}

type wrapper interface {
	Underlying() error
}

type causer interface {
	Cause() error
}
