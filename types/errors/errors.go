package errors

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// RootCodespace is the codespace for all errors defined in this package
const RootCodespace = "sdk"

// UndefinedCodespace when we explicitly declare no codespace
const UndefinedCodespace = "undefined"

var (
	// errInternal should never be exposed, but we reserve this code for non-specified errors
	errInternal = Register(UndefinedCodespace, 1, "internal")

	// ErrTxDecode is returned if we cannot parse a transaction
	ErrTxDecode = Register(RootCodespace, 2, "tx parse error")

	// ErrInvalidSequence is used the sequence number (nonce) is incorrect
	// for the signature
	ErrInvalidSequence = Register(RootCodespace, 3, "invalid sequence")

	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = Register(RootCodespace, 4, "unauthorized")

	// ErrInsufficientFunds is used when the account cannot pay requested amount.
	ErrInsufficientFunds = Register(RootCodespace, 5, "insufficient funds")

	// ErrUnknownRequest to doc
	ErrUnknownRequest = Register(RootCodespace, 6, "unknown request")

	// ErrInvalidAddress to doc
	ErrInvalidAddress = Register(RootCodespace, 7, "invalid address")

	// ErrInvalidPubKey to doc
	ErrInvalidPubKey = Register(RootCodespace, 8, "invalid pubkey")

	// ErrUnknownAddress to doc
	ErrUnknownAddress = Register(RootCodespace, 9, "unknown address")

	// ErrInvalidCoins to doc
	ErrInvalidCoins = Register(RootCodespace, 10, "invalid coins")

	// ErrOutOfGas to doc
	ErrOutOfGas = Register(RootCodespace, 11, "out of gas")

	// ErrMemoTooLarge to doc
	ErrMemoTooLarge = Register(RootCodespace, 12, "memo too large")

	// ErrInsufficientFee to doc
	ErrInsufficientFee = Register(RootCodespace, 13, "insufficient fee")

	// ErrTooManySignatures to doc
	ErrTooManySignatures = Register(RootCodespace, 14, "maximum number of signatures exceeded")

	// ErrNoSignatures to doc
	ErrNoSignatures = Register(RootCodespace, 15, "no signatures supplied")

	// ErrJSONMarshal defines an ABCI typed JSON marshalling error
	ErrJSONMarshal = Register(RootCodespace, 16, "failed to marshal JSON bytes")

	// ErrJSONUnmarshal defines an ABCI typed JSON unmarshalling error
	ErrJSONUnmarshal = Register(RootCodespace, 17, "failed to unmarshal JSON bytes")

	// ErrInvalidRequest defines an ABCI typed error where the request contains
	// invalid data.
	ErrInvalidRequest = Register(RootCodespace, 18, "invalid request")

	// ErrTxInMempoolCache defines an ABCI typed error where a tx already exists
	// in the mempool.
	ErrTxInMempoolCache = Register(RootCodespace, 19, "tx already in mempool")

	// ErrMempoolIsFull defines an ABCI typed error where the mempool is full.
	ErrMempoolIsFull = Register(RootCodespace, 20, "mempool is full")

	// ErrTxTooLarge defines an ABCI typed error where tx is too large.
	ErrTxTooLarge = Register(RootCodespace, 21, "tx too large")

	// ErrKeyNotFound defines an error when the key doesn't exist
	ErrKeyNotFound = Register(RootCodespace, 22, "key not found")

	// ErrWrongPassword defines an error when the key password is invalid.
	ErrWrongPassword = Register(RootCodespace, 23, "invalid account password")

	// ErrorInvalidSigner defines an error when the tx intended signer does not match the given signer.
	ErrorInvalidSigner = Register(RootCodespace, 24, "tx intended signer does not match the given signer")

	// ErrorInvalidGasAdjustment defines an error for an invalid gas adjustment
	ErrorInvalidGasAdjustment = Register(RootCodespace, 25, "invalid gas adjustment")

	// ErrInvalidHeight defines an error for an invalid height
	ErrInvalidHeight = Register(RootCodespace, 26, "invalid height")

	// ErrInvalidVersion defines a general error for an invalid version
	ErrInvalidVersion = Register(RootCodespace, 27, "invalid version")

	// ErrInvalidChainID defines an error when the chain-id is invalid.
	ErrInvalidChainID = Register(RootCodespace, 28, "invalid chain-id")

	// ErrInvalidType defines an error an invalid type.
	ErrInvalidType = Register(RootCodespace, 29, "invalid type")

	// ErrTxTimeoutHeight defines an error for when a tx is rejected out due to an
	// explicitly set timeout height.
	ErrTxTimeoutHeight = Register(RootCodespace, 30, "tx timeout height")

	// ErrUnknownExtensionOptions defines an error for unknown extension options.
	ErrUnknownExtensionOptions = Register(RootCodespace, 31, "unknown extension options")

	// ErrWrongSequence defines an error where the account sequence defined in
	// the signer info doesn't match the account's actual sequence number.
	ErrWrongSequence = Register(RootCodespace, 32, "incorrect account sequence")

	// ErrPackAny defines an error when packing a protobuf message to Any fails.
	ErrPackAny = Register(RootCodespace, 33, "failed packing protobuf message to Any")

	// ErrUnpackAny defines an error when unpacking a protobuf message from Any fails.
	ErrUnpackAny = Register(RootCodespace, 34, "failed unpacking protobuf message from Any")

	// ErrLogic defines an internal logic error, e.g. an invariant or assertion
	// that is violated. It is a programmer error, not a user-facing error.
	ErrLogic = Register(RootCodespace, 35, "internal logic error")

	// ErrConflict defines a conflict error, e.g. when two goroutines try to access
	// the same resource and one of them fails.
	ErrConflict = Register(RootCodespace, 36, "conflict")

	// ErrNotSupported is returned when we call a branch of a code which is currently not
	// supported.
	ErrNotSupported = Register(RootCodespace, 37, "feature not supported")

	// ErrNotFound defines an error when requested entity doesn't exist in the state.
	ErrNotFound = Register(RootCodespace, 38, "not found")

	// ErrIO should be used to wrap internal errors caused by external operation.
	// Examples: not DB domain error, file writing etc...
	ErrIO = Register(RootCodespace, 39, "Internal IO error")

	// ErrAppConfig defines an error occurred if min-gas-prices field in BaseConfig is empty.
	ErrAppConfig = Register(RootCodespace, 40, "error in app.toml")

	// ErrInvalidConcurrency defines an error occurred during concurrent execution
	ErrInvalidConcurrencyExecution = Register(RootCodespace, 41, "error during concurrent execution")

	// ErrAlreadyExists defines an error for which the tx failed checkTx because the node has already seen it before
	ErrAlreadyExists = Register(RootCodespace, 42, "error tx already exists")

	// ErrOCCAbort defines an error exncountered by a transaction when it encounters an OCC conflict resulting in an Abort
	ErrOCCAbort = Register(RootCodespace, 43, "occ abort")

	// ErrUnsupportedTxType defines an error for an unsupported transaction type (e.g., EIP-4844 transactions)
	ErrUnsupportedTxType = Register(RootCodespace, 44, "unsupported transaction type")

	// ErrEVMVMError defines an error for an evm vm error (eg. revert)
	ErrEVMVMError = Register(RootCodespace, 45, "evm reverted")

	// ErrInvalidRecipient defines an error for sending to disallowed recipients in bank
	ErrInvalidRecipient = Register(RootCodespace, 46, "invalid bank recipient")

	// ErrPanic is only set when we recover from a panic, so we know to
	// redact potentially sensitive system info
	ErrPanic = Register(UndefinedCodespace, 111222, "panic")
)

// Register returns an error instance that should be used as the base for
// creating error instances during runtime.
//
// Popular root errors are declared in this package, but extensions may want to
// declare custom codes. This function ensures that no error code is used
// twice. Attempt to reuse an error code results in panic.
//
// Use this function only during a program startup phase.
func Register(codespace string, code uint32, description string) *Error {
	// TODO - uniqueness is (codespace, code) combo
	if e := getUsed(codespace, code); e != nil {
		panic(fmt.Sprintf("error with code %d is already registered: %q", code, e.desc))
	}

	err := New(codespace, code, description)
	setUsed(err)

	return err
}

// usedCodes is keeping track of used codes to ensure their uniqueness. No two
// error instances should share the same (codespace, code) tuple.
var usedCodes = map[string]*Error{}

func errorID(codespace string, code uint32) string {
	return fmt.Sprintf("%s:%d", codespace, code)
}

func getUsed(codespace string, code uint32) *Error {
	return usedCodes[errorID(codespace, code)]
}

func setUsed(err *Error) {
	usedCodes[errorID(err.codespace, err.code)] = err
}

// ABCIError will resolve an error code/log from an abci result into
// an error message. If the code is registered, it will map it back to
// the canonical error, so we can do eg. ErrNotFound.Is(err) on something
// we get back from an external API.
//
// This should *only* be used in clients, not in the server side.
// The server (abci app / blockchain) should only refer to registered errors
func ABCIError(codespace string, code uint32, log string) error {
	if e := getUsed(codespace, code); e != nil {
		return Wrap(e, log)
	}
	// This is a unique error, will never match on .Is()
	// Use Wrap here to get a stack trace
	return Wrap(New(codespace, code, "unknown"), log)
}

// Error represents a root error.
//
// Weave framework is using root error to categorize issues. Each instance
// created during the runtime should wrap one of the declared root errors. This
// allows error tests and returning all errors to the client in a safe manner.
//
// All popular root errors are declared in this package. If an extension has to
// declare a custom root error, always use Register function to ensure
// error code uniqueness.
type Error struct {
	codespace string
	code      uint32
	desc      string
}

func New(codespace string, code uint32, desc string) *Error {
	return &Error{codespace: codespace, code: code, desc: desc}
}

func (e Error) Error() string {
	return e.desc
}

func (e Error) ABCICode() uint32 {
	return e.code
}

func (e Error) Codespace() string {
	return e.codespace
}

// Is check if given error instance is of a given kind/type. This involves
// unwrapping given error using the Cause method if available.
func (e *Error) Is(err error) bool {
	// Reflect usage is necessary to correctly compare with
	// a nil implementation of an error.
	if e == nil {
		return isNilErr(err)
	}

	for {
		if err == e {
			return true
		}

		// If this is a collection of errors, this function must return
		// true if at least one from the group match.
		if u, ok := err.(unpacker); ok {
			for _, er := range u.Unpack() {
				if e.Is(er) {
					return true
				}
			}
		}

		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {
			return false
		}
	}
}

// Wrap extends this error with an additional information.
// It's a handy function to call Wrap with sdk errors.
func (e *Error) Wrap(desc string) error { return Wrap(e, desc) }

// Wrapf extends this error with an additional information.
// It's a handy function to call Wrapf with sdk errors.
func (e *Error) Wrapf(desc string, args ...interface{}) error { return Wrapf(e, desc, args...) }

func isNilErr(err error) bool {
	// Reflect usage is necessary to correctly compare with
	// a nil implementation of an error.
	if err == nil {
		return true
	}
	if reflect.ValueOf(err).Kind() == reflect.Struct {
		return false
	}
	return reflect.ValueOf(err).IsNil()
}

// Wrap extends given error with an additional information.
//
// If the wrapped error does not provide ABCICode method (ie. stdlib errors),
// it will be labeled as internal error.
//
// If err is nil, this returns nil, avoiding the need for an if statement when
// wrapping a error returned at the end of a function
func Wrap(err error, description string) error {
	if err == nil {
		return nil
	}

	// If this error does not carry the stacktrace information yet, attach
	// one. This should be done only once per error at the lowest frame
	// possible (most inner wrap).
	if stackTrace(err) == nil {
		err = errors.WithStack(err)
	}

	return &wrappedError{
		parent: err,
		msg:    description,
	}
}

// Wrapf extends given error with an additional information.
//
// This function works like Wrap function with additional functionality of
// formatting the input as specified.
func Wrapf(err error, format string, args ...interface{}) error {
	desc := fmt.Sprintf(format, args...)
	return Wrap(err, desc)
}

type wrappedError struct {
	// This error layer description.
	msg string
	// The underlying error that triggered this one.
	parent error
}

func (e *wrappedError) Error() string {
	return fmt.Sprintf("%s: %s", e.msg, e.parent.Error())
}

func (e *wrappedError) Cause() error {
	return e.parent
}

// Is reports whether any error in e's chain matches a target.
func (e *wrappedError) Is(target error) bool {
	if e == target {
		return true
	}

	w := e.Cause()
	for {
		if w == target {
			return true
		}

		x, ok := w.(causer)
		if ok {
			w = x.Cause()
		}
		if x == nil {
			return false
		}
	}
}

// Unwrap implements the built-in errors.Unwrap
func (e *wrappedError) Unwrap() error {
	return e.parent
}

// Recover captures a panic and stop its propagation. If panic happens it is
// transformed into a ErrPanic instance and assigned to given error. Call this
// function using defer in order to work as expected.
func Recover(err *error) {
	if r := recover(); r != nil {
		*err = Wrapf(ErrPanic, "%v", r)
	}
}

// WithType is a helper to augment an error with a corresponding type message
func WithType(err error, obj interface{}) error {
	return Wrap(err, fmt.Sprintf("%T", obj))
}

// IsOf checks if a received error is caused by one of the target errors.
// It extends the errors.Is functionality to a list of errors.
func IsOf(received error, targets ...error) bool {
	for _, t := range targets {
		if errors.Is(received, t) {
			return true
		}
	}
	return false
}

// causer is an interface implemented by an error that supports wrapping. Use
// it to test if an error wraps another error instance.
type causer interface {
	Cause() error
}

type unpacker interface {
	Unpack() []error
}
