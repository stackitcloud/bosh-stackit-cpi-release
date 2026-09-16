package lib

import (
	"context"
	"encoding/json"
	"fmt"
)

/* RPCRequest holds the data provided by bosh / bosh cli via stdin
 * BOSH CPI RPC Request API¶
 * method [String]: Name of the CPI method. Example: create_vm.
 * arguments [Array]: Array of arguments that are specific to the CPI method.
 * context [Hash]: Additional information provided as a context of this execution.
 * ref: https://bosh.io/docs/cpi-api-rpc/#request
 */
type RPCRequest struct {
	Method     string          `json:"method"`                // Name of the CPI method
	Arguments  json.RawMessage `json:"arguments"`             // Array of arguments that are specific to the CPI method
	APIVersion int             `json:"api_version,omitempty"` // API version for the request
	Context    RPCContext      `json:"context,omitzero"`      // Context for the request
}
type RPCContext struct {
	DirectorUUID string    `json:"director_uuid"` // UUID of the DirectorUUID
	RequestID    string    `json:"request_id"`    // Request ID for the RPCContext
	VM           VMContext `json:"vm"`
	Config       Config    `json:"stackit"` // if we are using multi cpi in the director and the vm is supposed to be created by a non default cpi, the config for the additional CPI will be provided by the call via the context
}

/* RPCResponse is used to communicate the result of the operation to the bosh director
 * result [Null or simple values]: Single return value. It must be null if error is returned.
 * error [Null or hash]: Occurred error. It must be null if result is returned.
 * type [String]: Type of the error.
 * message [String]: Description of the error.
 * ok_to_retry [Boolean]: Indicates whether callee should try calling the method again without changing any of the arguments.
 * log [String]: Additional information that may be useful for auditing, debugging and understanding what actions CPI took while executing a method. Typically includes info and debug logs, error backtraces.
 * ref: https://bosh.io/docs/cpi-rpc.html#response
 */
type RPCResponse struct {
	Result any       `json:"result"` // Single return value. It must be null if error is returned.
	Error  *RPCError `json:"error"`  // Occurred error. It must be null if result is returned.
	Log    string    `json:"log"`    // Additional information that may be useful for auditing, debugging and understanding what actions CPI took while executing a method. Typically includes info and debug logs, error backtraces.
}

type RPCError struct {
	Type      string `json:"type,omitempty"`        // Type of the error. Omitted if there is no error.
	Message   string `json:"message,omitempty"`     // Description of the error. Omitted if there is no error.
	OkToRetry bool   `json:"ok_to_retry,omitempty"` // Indicates whether callee should try calling the method again without changing any of the arguments.
}

func (r *RPCResponse) String() string {
	bytes, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{
			"error": {
			  "type":"unknown",
				"Message": "failed to unmarshal response: '%s'"
			},
			"result": false,
			"log": "%s"
		}`, err, err)
	}
	return string(bytes)
}

func (r *RPCRequest) GetLoggingContext() context.Context {
	ctx := context.WithValue(context.Background(), requestIDKey, r.Context.RequestID)
	ctx = context.WithValue(ctx, cpiMethodKey, r.Method)
	ctx = context.WithValue(ctx, directorUUIDKey, r.Context.DirectorUUID)
	return ctx
}
