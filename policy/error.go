package policy

import "fmt"

type QueryMissingPathConfigError struct {
	Key string
}

func (e *QueryMissingPathConfigError) Error() string {
	return fmt.Sprintf("key %s missing from path config supplied to the policy agent client.", e.Key)
}

type QueryMarshallError struct {
	Err error
}

func (e *QueryMarshallError) Error() string {
	return fmt.Sprintf(
		"an error occurred while attempting to marshall a query for the policy engine: %s.",
		e.Err,
	)
}

type DecisionUnmarshallError struct {
	Err error
}

func (e *DecisionUnmarshallError) Error() string {
	return fmt.Sprintf(
		"an error occurred while attempting to unmarshall a decision from the policy engine: %s.",
		e.Err,
	)
}

type DecisionPayloadError struct {
	Msg string
}

func (e *DecisionPayloadError) Error() string {
	return fmt.Sprintf(
		"there was a problem with the decision payload: %s.",
		e.Msg,
	)
}

type QueryRequestCreationError struct {
	Err error
}

func (e *QueryRequestCreationError) Error() string {
	return fmt.Sprintf(
		"an error occurred while constructing a query request for the policy engine: %s.",
		e.Err,
	)
}

type QueryRequestError struct {
	Err error
}

func (e *QueryRequestError) Error() string {
	return fmt.Sprintf(
		"an error occurred while performing a query request to the policy engine: %s.",
		e.Err,
	)
}

type QueryResponseReadingError struct {
	Err error
}

func (e *QueryResponseReadingError) Error() string {
	return fmt.Sprintf(
		"an error occurred while reading a response from the policy engine: %s.",
		e.Err,
	)
}
