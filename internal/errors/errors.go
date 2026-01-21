package errors

import "unsafe"
import "errors"

type compoundError struct {
	err    error
	causes []error
}

func CausedBy(main error, causes ...error) error {
	n := 0
	for _, err := range causes {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return main
	}
	e := &compoundError{
		err:    main,
		causes: make([]error, 0, n),
	}
	for _, err := range causes {
		if err != nil {
			e.causes = append(e.causes, err)
		}
	}
	return e
}

func (e *compoundError) Error() string {
	if len(e.causes) < 1 {
		return e.err.Error()
	}

	b := []byte(e.err.Error())
	for _, err := range e.causes {
		b = append(b, "\n\t"...)
		b = append(b, err.Error()...)
	}
	return unsafe.String(&b[0], len(b))
}

func (e *compoundError) Causes() []error {
  return e.causes
}

func (e *compoundError) Is(target error) bool {
	return errors.Is(e.err, target)
}

func (e *compoundError) Unwrap() []error {
	return e.causes
}
